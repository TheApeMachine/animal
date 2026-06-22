package session

import (
	"fmt"
	"strings"

	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
)

const (
	metadataGoalID      = "goal_id"
	metadataLeasePrefix = "lease_prefix"
)

var observationStructuredOutput = ai.ObservationStructuredOutput(true)

/*
RunTask clones the agent for an A2A task and executes one interactive cycle.
*/
func (session *Session) RunTask(task a2a.Task) (Result, error) {
	if err := task.Validate(); err != nil {
		return Result{Status: StatusFailed}, err
	}

	participant := session.agent.Participant()
	if participant == nil {
		return Result{Status: StatusFailed}, errnie.Err(
			errnie.Validation,
			"session task run requires swarm participant",
			nil,
		)
	}

	if err := session.claimTaskLease(participant, task); err != nil {
		return session.failTask(participant, task, err)
	}

	defer session.releaseTaskLease(participant, task)

	if err := participant.StartTask(task.ID, "started"); err != nil {
		return Result{Status: StatusFailed}, err
	}

	result, err := session.cloneForTask(task)
	if err != nil {
		return session.failTask(participant, task, err)
	}

	if err := session.observeTask(participant, task, result); err != nil {
		if reportErr := participant.ReportSignal(
			swarm.SignalFriction,
			taskGoalID(task),
			task.ID,
			"observation pass failed",
			err.Error(),
		); reportErr != nil {
			return result, reportErr
		}
	}

	if err := participant.CompleteTask(task.ID, result.Assistant.Content); err != nil {
		return result, err
	}

	if err := participant.ReportMetric(
		taskGoalID(task),
		task.ID,
		"task_completed",
		1,
		true,
		result.Assistant.Content,
	); err != nil {
		return result, err
	}

	return result, nil
}

func (session *Session) cloneForTask(task a2a.Task) (Result, error) {
	clone, err := session.agent.CloneWithTask(session.ctx, task)
	if err != nil {
		return Result{Status: StatusFailed}, err
	}

	taskSession, err := NewSession(
		session.ctx,
		clone,
		session.streamer,
		session.bridge,
		session.params,
	)

	if err != nil {
		return Result{Status: StatusFailed}, err
	}

	return taskSession.Cycle()
}

func (session *Session) observeTask(
	participant *swarm.Participant,
	task a2a.Task,
	result Result,
) error {
	system, agentCtx, err := session.agent.ObservationContext(
		session.ctx,
		session.observationRequest(participant, task, result),
	)

	if err != nil {
		return err
	}

	params := session.params.Clone().
		WithStructuredOutput(&observationStructuredOutput)
	params.WithTemperature(0)

	var builder strings.Builder

	err = session.streamer.StreamWithSink(
		system,
		agentCtx,
		params,
		func(delta string) error {
			builder.WriteString(delta)

			return nil
		},
	)

	if err != nil {
		return err
	}

	observation, err := ai.ParseObservation(builder.String())
	if err != nil {
		return err
	}

	return session.agent.PublishObservation(observation)
}

func (session *Session) observationRequest(
	participant *swarm.Participant,
	task a2a.Task,
	result Result,
) ai.ObservationRequest {
	messages := append([]provider.Message(nil), session.agent.Context.Messages...)
	messages = append(messages, result.Prompt, result.Assistant)

	return ai.ObservationRequest{
		GoalID:        taskGoalID(task),
		TaskID:        task.ID,
		Instruction:   task.Instruction(),
		Prompt:        result.Prompt,
		Assistant:     result.Assistant,
		RecentSignals: participant.View().RecentSignals(),
		Messages:      messages,
	}
}

func (session *Session) claimTaskLease(
	participant *swarm.Participant,
	task a2a.Task,
) error {
	prefix := taskMetadata(task, metadataLeasePrefix)
	if prefix == "" {
		return nil
	}

	if err := participant.TryClaim(prefix); err != nil {
		participant.ReportSignal(
			swarm.SignalBlocker,
			taskGoalID(task),
			task.ID,
			"lease unavailable",
			err.Error(),
		)

		return err
	}

	return nil
}

func (session *Session) releaseTaskLease(
	participant *swarm.Participant,
	task a2a.Task,
) {
	prefix := taskMetadata(task, metadataLeasePrefix)
	if prefix == "" {
		return
	}

	if err := participant.Release(prefix); err != nil {
		participant.ReportSignal(
			swarm.SignalFriction,
			taskGoalID(task),
			task.ID,
			"lease release failed",
			err.Error(),
		)
	}
}

func (session *Session) failTask(
	participant *swarm.Participant,
	task a2a.Task,
	cause error,
) (Result, error) {
	reason := fmt.Sprintf("%v", cause)

	if err := participant.FailTask(task.ID, reason); err != nil {
		return Result{Status: StatusFailed}, err
	}

	return Result{Status: StatusFailed}, cause
}

func taskGoalID(task a2a.Task) string {
	return taskMetadata(task, metadataGoalID)
}

func taskMetadata(task a2a.Task, key string) string {
	value, ok := task.Metadata[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}
