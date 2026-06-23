package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
)

func (workflow *Workflow) runAgent(
	ctx context.Context,
	agent workflowAgent,
	registry *swarm.Registry,
) (WorkflowAgentResult, error) {
	result := workflow.agentResult(agent)

	spawned, err := NewAgent(
		ctx,
		workflow.pool,
		agent.slot.Persona,
		result.Name,
		registry,
		agent.slot.LeasePrefixes,
	)

	if err != nil {
		return workflow.failAgent(result, nil, err)
	}

	result.AgentID = spawned.ID
	participant := spawned.Participant()

	if participant == nil {
		err = errnie.Err(errnie.Validation, "workflow participant is required", nil)
		return workflow.failAgent(result, nil, err)
	}

	if err := workflow.startTask(participant, agent, result); err != nil {
		return workflow.failAgent(result, participant, err)
	}

	if agent.requireLeases {
		if err := workflow.claim(participant, agent, &result); err != nil {
			return workflow.failAgent(result, participant, err)
		}
	}

	if err := workflow.release(participant, result.Claimed); err != nil {
		return workflow.failAgent(result, participant, err)
	}

	if err := participant.CompleteTask(result.TaskID, "workflow coordination completed"); err != nil {
		return workflow.failAgent(result, participant, err)
	}

	result.Status = WorkflowStatusCompleted

	return result, nil
}

func (workflow *Workflow) startTask(
	participant *swarm.Participant,
	agent workflowAgent,
	result WorkflowAgentResult,
) error {
	_, err := participant.SubmitTask(
		result.TaskID,
		workflow.instruction(agent),
		workflow.metadata(agent),
	)

	if err != nil {
		return err
	}

	return participant.StartTask(result.TaskID, "workflow step started")
}

func (workflow *Workflow) claim(
	participant *swarm.Participant,
	agent workflowAgent,
	result *WorkflowAgentResult,
) error {
	for _, prefix := range agent.slot.LeasePrefixes {
		if err := participant.TryClaim(prefix); err != nil {
			message := err.Error()

			if err := workflow.release(participant, result.Claimed); err != nil {
				return errnie.Err(errnie.Validation, message+"; release failed", err)
			}

			return errnie.Err(errnie.Validation, message, nil)
		}

		result.Claimed = append(result.Claimed, prefix)
	}

	return nil
}

func (workflow *Workflow) release(
	participant *swarm.Participant,
	prefixes []string,
) error {
	for _, prefix := range prefixes {
		if err := participant.Release(prefix); err != nil {
			return err
		}
	}

	return nil
}

func (workflow *Workflow) failAgent(
	result WorkflowAgentResult,
	participant *swarm.Participant,
	err error,
) (WorkflowAgentResult, error) {
	result.Status = WorkflowStatusFailed
	result.Error = err.Error()

	if participant == nil || strings.TrimSpace(result.TaskID) == "" {
		return result, err
	}

	if _, ok := participant.View().Task(result.TaskID); !ok {
		return result, err
	}

	err = participant.FailTask(result.TaskID, result.Error)

	if err != nil {
		return result, err
	}

	return result, errnie.Err(errnie.Validation, result.Error, nil)
}

func (workflow *Workflow) stepAgents(
	definition config.WorkflowYAML,
	step config.WorkflowStepYAML,
	stepIndex int,
) []workflowAgent {
	if workflow.stepMode(step) == "conversation" {
		return workflow.conversationAgents(step, stepIndex)
	}

	return workflow.workAgents(definition, step, stepIndex)
}

func (workflow *Workflow) workAgents(
	definition config.WorkflowYAML,
	step config.WorkflowStepYAML,
	stepIndex int,
) []workflowAgent {
	agents := make([]workflowAgent, 0)
	requireLeases := definition.FileLeasing && step.RequireFileLeases

	for slotIndex, slot := range step.Slots {
		for replicaIndex := range slot.Replicas {
			agents = append(agents, workflowAgent{
				step:          step,
				slot:          slot,
				stepIndex:     stepIndex,
				slotIndex:     slotIndex,
				replicaIndex:  replicaIndex,
				requireLeases: requireLeases && !slot.ReadOnlyObserver,
			})
		}
	}

	return agents
}

func (workflow *Workflow) conversationAgents(
	step config.WorkflowStepYAML,
	stepIndex int,
) []workflowAgent {
	agents := make([]workflowAgent, 0, len(step.Conversation.Speakers))

	for speakerIndex, speaker := range step.Conversation.Speakers {
		agents = append(agents, workflowAgent{
			step:      step,
			stepIndex: stepIndex,
			slotIndex: speakerIndex,
			slot: config.WorkflowSlotYAML{
				Persona:  speaker.Persona,
				Replicas: 1,
			},
		})
	}

	return agents
}

func (workflow *Workflow) agentResult(agent workflowAgent) WorkflowAgentResult {
	return WorkflowAgentResult{
		Name:     workflow.agentName(agent),
		Role:     agent.slot.Persona,
		TaskID:   workflow.taskID(agent),
		ReadOnly: agent.slot.ReadOnlyObserver,
		Prefixes: append([]string(nil), agent.slot.LeasePrefixes...),
		Claimed:  make([]string, 0, len(agent.slot.LeasePrefixes)),
		Status:   WorkflowStatusFailed,
	}
}

func (workflow *Workflow) failedAgentResult(
	agent workflowAgent,
	message string,
) WorkflowAgentResult {
	result := workflow.agentResult(agent)
	result.Error = message

	return result
}

func (workflow *Workflow) instruction(agent workflowAgent) string {
	if strings.TrimSpace(agent.step.Description) != "" {
		return agent.step.Description
	}

	return fmt.Sprintf("Execute workflow step %s.", agent.step.ID)
}

func (workflow *Workflow) metadata(agent workflowAgent) map[string]any {
	return map[string]any{
		"step_id":       agent.step.ID,
		"step_index":    agent.stepIndex,
		"slot_index":    agent.slotIndex,
		"replica_index": agent.replicaIndex,
		"read_only":     agent.slot.ReadOnlyObserver,
	}
}

func (workflow *Workflow) jobID(agent workflowAgent) string {
	return fmt.Sprintf(
		"workflow:%d:%s:%d:%d",
		time.Now().UnixNano(),
		agent.step.ID,
		agent.slotIndex,
		agent.replicaIndex,
	)
}

func (workflow *Workflow) taskID(agent workflowAgent) string {
	return fmt.Sprintf(
		"%s:%s:%d:%d",
		agent.step.ID,
		agent.slot.Persona,
		agent.slotIndex,
		agent.replicaIndex,
	)
}

func (workflow *Workflow) agentName(agent workflowAgent) string {
	return fmt.Sprintf(
		"%s-%d-%d",
		agent.slot.Persona,
		agent.slotIndex+1,
		agent.replicaIndex+1,
	)
}
