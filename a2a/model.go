package a2a

import (
	"fmt"
	"strings"

	"github.com/theapemachine/errnie"
)

/*
Role identifies the speaker of an A2A message.
*/
type Role string

const (
	RoleUser  Role = "user"
	RoleAgent Role = "agent"
)

/*
TaskState mirrors the A2A task lifecycle enum.
*/
type TaskState string

const (
	TaskStateUnspecified   TaskState = "TASK_STATE_UNSPECIFIED"
	TaskStateSubmitted     TaskState = "TASK_STATE_SUBMITTED"
	TaskStateWorking       TaskState = "TASK_STATE_WORKING"
	TaskStateInputRequired TaskState = "TASK_STATE_INPUT_REQUIRED"
	TaskStateCompleted     TaskState = "TASK_STATE_COMPLETED"
	TaskStateFailed        TaskState = "TASK_STATE_FAILED"
	TaskStateCanceled      TaskState = "TASK_STATE_CANCELED"
)

/*
Part is one A2A content part. Presence of text, data, raw, or url is the discriminator.
*/
type Part struct {
	Text      string         `json:"text,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Raw       string         `json:"raw,omitempty"`
	URL       string         `json:"url,omitempty"`
	Filename  string         `json:"filename,omitempty"`
	MediaType string         `json:"mediaType,omitempty"`
}

/*
Message is the A2A communication unit used to start or continue tasks.
*/
type Message struct {
	MessageID        string         `json:"messageId,omitempty"`
	TaskID           string         `json:"taskId,omitempty"`
	ContextID        string         `json:"contextId,omitempty"`
	Role             Role           `json:"role"`
	Parts            []Part         `json:"parts"`
	ReferenceTaskIDs []string       `json:"referenceTaskIds,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

/*
Artifact is an A2A task output container.
*/
type Artifact struct {
	ArtifactID  string         `json:"artifactId,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parts       []Part         `json:"parts,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

/*
TaskStatus is the current A2A task state plus an optional status message.
*/
type TaskStatus struct {
	State     TaskState `json:"state"`
	Message   *Message  `json:"message,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
}

/*
Task is the A2A unit of action.
*/
type Task struct {
	ID        string         `json:"id"`
	ContextID string         `json:"contextId,omitempty"`
	Status    TaskStatus     `json:"status"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
	History   []Message      `json:"history,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

/*
TaskStatusUpdateEvent is the streaming status event for a task.
*/
type TaskStatusUpdateEvent struct {
	TaskID    string     `json:"taskId"`
	ContextID string     `json:"contextId,omitempty"`
	Status    TaskStatus `json:"status"`
	Final     bool       `json:"final,omitempty"`
}

/*
TaskArtifactUpdateEvent is the streaming artifact event for a task.
*/
type TaskArtifactUpdateEvent struct {
	TaskID    string   `json:"taskId"`
	ContextID string   `json:"contextId,omitempty"`
	Artifact  Artifact `json:"artifact"`
	Append    bool     `json:"append,omitempty"`
	LastChunk bool     `json:"lastChunk,omitempty"`
}

/*
AgentCapabilities declares A2A operation support.
*/
type AgentCapabilities struct {
	Streaming              bool `json:"streaming,omitempty"`
	PushNotifications      bool `json:"pushNotifications,omitempty"`
	StateTransitionHistory bool `json:"stateTransitionHistory,omitempty"`
	ExtendedAgentCard      bool `json:"extendedAgentCard,omitempty"`
}

/*
AgentCard is the A2A discovery document.
*/
type AgentCard struct {
	Name               string            `json:"name"`
	Description        string            `json:"description,omitempty"`
	URL                string            `json:"url"`
	ProtocolVersion    string            `json:"protocolVersion,omitempty"`
	ProtocolVersions   []string          `json:"protocolVersions,omitempty"`
	Capabilities       AgentCapabilities `json:"capabilities"`
	DefaultInputModes  []string          `json:"defaultInputModes,omitempty"`
	DefaultOutputModes []string          `json:"defaultOutputModes,omitempty"`
}

/*
Validate checks that a part has exactly one payload discriminator.
*/
func (part Part) Validate() error {
	count := 0

	if strings.TrimSpace(part.Text) != "" {
		count++
	}

	if len(part.Data) > 0 {
		count++
	}

	if strings.TrimSpace(part.Raw) != "" {
		count++
	}

	if strings.TrimSpace(part.URL) != "" {
		count++
	}

	if count != 1 {
		return errnie.Err(
			errnie.Validation,
			"a2a part requires exactly one payload",
			nil,
		)
	}

	return nil
}

/*
Validate checks required A2A message fields.
*/
func (message Message) Validate() error {
	if !validRole(message.Role) {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("a2a message role %q is invalid", message.Role),
			nil,
		)
	}

	if len(message.Parts) == 0 {
		return errnie.Err(errnie.Validation, "a2a message parts are required", nil)
	}

	for _, part := range message.Parts {
		if err := part.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Text joins text parts from the message in order.
*/
func (message Message) Text() string {
	values := make([]string, 0, len(message.Parts))

	for _, part := range message.Parts {
		if strings.TrimSpace(part.Text) == "" {
			continue
		}

		values = append(values, part.Text)
	}

	return strings.Join(values, "\n")
}

/*
Validate checks required A2A artifact fields.
*/
func (artifact Artifact) Validate() error {
	for _, part := range artifact.Parts {
		if err := part.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Validate checks required A2A task status fields.
*/
func (status TaskStatus) Validate() error {
	if !validTaskState(status.State) {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("a2a task state %q is invalid", status.State),
			nil,
		)
	}

	if status.Message == nil {
		return nil
	}

	return status.Message.Validate()
}

/*
Validate checks required A2A task fields.
*/
func (task Task) Validate() error {
	if strings.TrimSpace(task.ID) == "" {
		return errnie.Err(errnie.Validation, "a2a task id is required", nil)
	}

	if err := task.Status.Validate(); err != nil {
		return err
	}

	for _, message := range task.History {
		if err := message.Validate(); err != nil {
			return err
		}
	}

	for _, artifact := range task.Artifacts {
		if err := artifact.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Instruction returns the latest user text available on the task.
*/
func (task Task) Instruction() string {
	for index := len(task.History) - 1; index >= 0; index-- {
		if task.History[index].Role != RoleUser {
			continue
		}

		return task.History[index].Text()
	}

	if task.Status.Message == nil {
		return ""
	}

	return task.Status.Message.Text()
}

/*
Validate checks required streaming status event fields.
*/
func (event TaskStatusUpdateEvent) Validate() error {
	if strings.TrimSpace(event.TaskID) == "" {
		return errnie.Err(errnie.Validation, "a2a task status event task id is required", nil)
	}

	return event.Status.Validate()
}

/*
Validate checks required streaming artifact event fields.
*/
func (event TaskArtifactUpdateEvent) Validate() error {
	if strings.TrimSpace(event.TaskID) == "" {
		return errnie.Err(errnie.Validation, "a2a task artifact event task id is required", nil)
	}

	return event.Artifact.Validate()
}

/*
Validate checks required A2A agent card fields.
*/
func (card AgentCard) Validate() error {
	if strings.TrimSpace(card.Name) == "" {
		return errnie.Err(errnie.Validation, "a2a agent card name is required", nil)
	}

	if strings.TrimSpace(card.URL) == "" {
		return errnie.Err(errnie.Validation, "a2a agent card url is required", nil)
	}

	return nil
}

func validRole(role Role) bool {
	switch role {
	case RoleUser, RoleAgent:
		return true
	default:
		return false
	}
}

func validTaskState(state TaskState) bool {
	switch state {
	case TaskStateUnspecified,
		TaskStateSubmitted,
		TaskStateWorking,
		TaskStateInputRequired,
		TaskStateCompleted,
		TaskStateFailed,
		TaskStateCanceled:
		return true
	default:
		return false
	}
}
