package swarm

import (
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/errnie"
)

/*
SubmitTask broadcasts a new A2A task with participant metadata.
*/
func (participant *Participant) SubmitTask(
	taskID string,
	instruction string,
	metadata map[string]any,
) (a2a.Task, error) {
	if strings.TrimSpace(taskID) == "" {
		return a2a.Task{}, errnie.Err(errnie.Validation, "swarm task id is required", nil)
	}

	if strings.TrimSpace(instruction) == "" {
		return a2a.Task{}, errnie.Err(errnie.Validation, "swarm task instruction is required", nil)
	}

	message := participant.taskMessage(taskID, instruction, metadata)
	task := a2a.Task{
		ID: taskID,
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateSubmitted,
			Message:   &message,
			Timestamp: participant.timestamp(),
		},
		History:  []a2a.Message{message},
		Metadata: participant.metadata(metadata),
	}

	if err := participant.PublishTask(task); err != nil {
		return a2a.Task{}, err
	}

	return task, nil
}

/*
StartTask marks a task as working.
*/
func (participant *Participant) StartTask(taskID string, note string) error {
	event, err := participant.taskStatusEvent(
		taskID,
		a2a.TaskStateWorking,
		note,
		false,
	)

	if err != nil {
		return err
	}

	return participant.PublishTaskStatus(event)
}

/*
CompleteTask marks a task as completed.
*/
func (participant *Participant) CompleteTask(taskID string, note string) error {
	event, err := participant.taskStatusEvent(
		taskID,
		a2a.TaskStateCompleted,
		note,
		true,
	)

	if err != nil {
		return err
	}

	return participant.PublishTaskStatus(event)
}

/*
FailTask marks a task as failed.
*/
func (participant *Participant) FailTask(taskID string, reason string) error {
	event, err := participant.taskStatusEvent(
		taskID,
		a2a.TaskStateFailed,
		reason,
		true,
	)

	if err != nil {
		return err
	}

	return participant.PublishTaskStatus(event)
}

/*
CancelTask marks a task as canceled.
*/
func (participant *Participant) CancelTask(taskID string, reason string) error {
	event, err := participant.taskStatusEvent(
		taskID,
		a2a.TaskStateCanceled,
		reason,
		true,
	)

	if err != nil {
		return err
	}

	return participant.PublishTaskStatus(event)
}

/*
PublishTask broadcasts an A2A task to the swarm.
*/
func (participant *Participant) PublishTask(task a2a.Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	if err := participant.view.MergeTask(task); err != nil {
		return err
	}

	return participant.mesh.PublishValue(participant.actorID, MessageTypeTask, task)
}

/*
PublishTaskStatus broadcasts an A2A streaming status update to the swarm.
*/
func (participant *Participant) PublishTaskStatus(event a2a.TaskStatusUpdateEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	if err := participant.view.MergeTaskStatus(event); err != nil {
		return err
	}

	return participant.mesh.PublishValue(participant.actorID, MessageTypeTaskStatus, event)
}

func (participant *Participant) taskStatusEvent(
	taskID string,
	state a2a.TaskState,
	text string,
	final bool,
) (a2a.TaskStatusUpdateEvent, error) {
	if strings.TrimSpace(taskID) == "" {
		return a2a.TaskStatusUpdateEvent{}, errnie.Err(
			errnie.Validation,
			"swarm task id is required",
			nil,
		)
	}

	if strings.TrimSpace(text) == "" {
		return a2a.TaskStatusUpdateEvent{}, errnie.Err(
			errnie.Validation,
			"swarm task status text is required",
			nil,
		)
	}

	message := participant.taskMessage(taskID, text, nil)

	return a2a.TaskStatusUpdateEvent{
		TaskID: taskID,
		Status: a2a.TaskStatus{
			State:     state,
			Message:   &message,
			Timestamp: participant.timestamp(),
		},
		Final: final,
	}, nil
}

func (participant *Participant) taskMessage(
	taskID string,
	text string,
	metadata map[string]any,
) a2a.Message {
	timestamp := time.Now().UTC().UnixNano()

	return a2a.Message{
		MessageID: fmt.Sprintf("%s:%s:%d", participant.actorID, taskID, timestamp),
		TaskID:    taskID,
		Role:      a2a.RoleAgent,
		Parts: []a2a.Part{
			{Text: text},
		},
		Metadata: participant.metadata(metadata),
	}
}

func (participant *Participant) metadata(metadata map[string]any) map[string]any {
	out := make(map[string]any, len(metadata)+3)
	maps.Copy(out, metadata)
	out["actor_id"] = participant.actorID
	out["actor_name"] = participant.actorName
	out["role"] = participant.role

	return out
}

func (participant *Participant) timestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
