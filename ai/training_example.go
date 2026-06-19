package ai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
)

/*
FineTuneMessage is one OpenAI chat fine-tuning message.
*/
type FineTuneMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

/*
FineTuneExample is one JSONL record for periodic fine-tuning.
*/
type FineTuneExample struct {
	Messages []FineTuneMessage `json:"messages"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

/*
FineTuneExample builds one fine-tuning record from the current agent trace.
*/
func (agent *Agent) FineTuneExample(metric swarm.Metric) (FineTuneExample, error) {
	if agent == nil {
		return FineTuneExample{}, errnie.Err(errnie.Validation, "training record agent is required", nil)
	}

	if err := metric.Validate(); err != nil {
		return FineTuneExample{}, err
	}

	if !metric.Success {
		return FineTuneExample{}, errnie.Err(errnie.Validation, "training metric must be successful", nil)
	}

	if metric.ActorID != agent.ID {
		return FineTuneExample{}, errnie.Err(errnie.Validation, "training metric actor must match agent", nil)
	}

	messages := make([]FineTuneMessage, 0, len(agent.Context.Messages)+1)
	messages = append(messages, FineTuneMessage{
		Role:    "system",
		Content: agent.System,
	})

	for _, message := range agent.Context.Messages {
		fineTuneMessage, err := newFineTuneMessage(message)
		if err != nil {
			return FineTuneExample{}, err
		}

		messages = append(messages, fineTuneMessage)
	}

	return FineTuneExample{
		Messages: messages,
		Metadata: map[string]string{
			"actor_id": metric.ActorID,
			"goal_id":  metric.GoalID,
			"task_id":  metric.TaskID,
			"metric":   metric.Name,
			"score":    fmt.Sprintf("%.6f", metric.Score),
			"success":  strconv.FormatBool(metric.Success),
		},
	}, nil
}

/*
JSONL serializes one fine-tuning example as one newline-terminated JSONL record.
*/
func (example FineTuneExample) JSONL() ([]byte, error) {
	if err := example.Validate(); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(example)
	if err != nil {
		return nil, errnie.Err(errnie.Validation, "training example marshal failed", err)
	}

	return append(payload, '\n'), nil
}

/*
Validate checks the JSONL fine-tuning example shape.
*/
func (example FineTuneExample) Validate() error {
	if len(example.Messages) == 0 {
		return errnie.Err(errnie.Validation, "training example messages are required", nil)
	}

	for _, message := range example.Messages {
		if err := message.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Validate checks one fine-tuning message.
*/
func (message FineTuneMessage) Validate() error {
	if strings.TrimSpace(message.Role) == "" {
		return errnie.Err(errnie.Validation, "training example role is required", nil)
	}

	if strings.TrimSpace(message.Content) == "" {
		return errnie.Err(errnie.Validation, "training example content is required", nil)
	}

	return nil
}

func newFineTuneMessage(message provider.Message) (FineTuneMessage, error) {
	role := strings.ToLower(strings.TrimSpace(message.Role))

	switch role {
	case "system", "user", "assistant":
	default:
		return FineTuneMessage{}, errnie.Err(
			errnie.Validation,
			fmt.Sprintf("training message role %q is unsupported", message.Role),
			nil,
		)
	}

	if strings.TrimSpace(message.Content) == "" {
		return FineTuneMessage{}, errnie.Err(
			errnie.Validation,
			"training message content is required",
			nil,
		)
	}

	return FineTuneMessage{
		Role:    role,
		Content: message.Content,
	}, nil
}
