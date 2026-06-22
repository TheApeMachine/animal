package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
)

const observationTemplatePath = "ai.prompt.template.observation"

/*
ObservationRequest is the temporary context used between task generations.
*/
type ObservationRequest struct {
	GoalID        string             `json:"goal_id"`
	TaskID        string             `json:"task_id"`
	Instruction   string             `json:"instruction"`
	Prompt        provider.Message   `json:"prompt"`
	Assistant     provider.Message   `json:"assistant"`
	RecentSignals []swarm.Signal     `json:"recent_signals"`
	Messages      []provider.Message `json:"messages"`
}

/*
Observation is the structured output produced by the swarm observation pass.
*/
type Observation struct {
	Signals []ObservationSignal `json:"signals"`
}

/*
ObservationSignal is one model-requested swarm signal.
*/
type ObservationSignal struct {
	Kind    swarm.SignalKind `json:"kind"`
	GoalID  string           `json:"goal_id"`
	TaskID  string           `json:"task_id"`
	Summary string           `json:"summary"`
	Detail  string           `json:"detail"`
}

/*
ObservationStructuredOutput returns the schema used by the observation pass.
*/
func ObservationStructuredOutput(strict bool) provider.StructuredOutput {
	return provider.StructuredOutput{
		Name:        "swarm_observation",
		Description: "Swarm friction, quality, blocker, and opportunity observations.",
		Strict:      strict,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"signals": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"kind": map[string]any{
								"type": "string",
								"enum": []string{
									string(swarm.SignalFriction),
									string(swarm.SignalQuality),
									string(swarm.SignalBlocker),
									string(swarm.SignalOpportunity),
								},
							},
							"goal_id": map[string]any{"type": "string"},
							"task_id": map[string]any{"type": "string"},
							"summary": map[string]any{"type": "string"},
							"detail":  map[string]any{"type": "string"},
						},
						"required": []string{
							"kind",
							"goal_id",
							"task_id",
							"summary",
							"detail",
						},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"signals"},
			"additionalProperties": false,
		},
	}
}

/*
ObservationContext builds the temporary context for the between-generation observation pass.
*/
func (agent *Agent) ObservationContext(
	ctx context.Context,
	request ObservationRequest,
) (string, *provider.Context, error) {
	if ctx == nil {
		return "", nil, errnie.Err(errnie.Validation, "observation context is required", nil)
	}

	system, err := agent.observationSystem()
	if err != nil {
		return "", nil, err
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return "", nil, errnie.Err(errnie.Validation, "observation request marshal failed", err)
	}

	agentCtx := provider.NewContext(ctx)
	if err := agentCtx.Append(provider.Message{
		Role:    "user",
		Content: string(payload),
	}); err != nil {
		return "", nil, err
	}

	return system, agentCtx, nil
}

/*
ParseObservation decodes provider structured output into an Observation.
*/
func ParseObservation(payload string) (Observation, error) {
	var observation Observation

	if strings.TrimSpace(payload) == "" {
		return Observation{}, errnie.Err(errnie.Validation, "observation payload is required", nil)
	}

	if err := json.Unmarshal([]byte(payload), &observation); err != nil {
		return Observation{}, errnie.Err(errnie.Validation, "observation payload decode failed", err)
	}

	if err := observation.Validate(); err != nil {
		return Observation{}, err
	}

	return observation, nil
}

/*
Validate checks one observation output.
*/
func (observation Observation) Validate() error {
	for _, signal := range observation.Signals {
		if err := signal.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Validate checks one observation signal.
*/
func (signal ObservationSignal) Validate() error {
	swarmSignal := swarm.NewSignalAt(
		signal.Kind,
		"observation",
		"observation",
		"observer",
		time.Now(),
	)
	swarmSignal.Summary = signal.Summary

	return swarmSignal.Validate()
}

/*
PublishObservation broadcasts all signals requested by an observation pass.
*/
func (agent *Agent) PublishObservation(observation Observation) error {
	if agent.participant == nil {
		return errnie.Err(errnie.Validation, "observation publish requires swarm participant", nil)
	}

	if err := observation.Validate(); err != nil {
		return err
	}

	for _, signal := range observation.Signals {
		if err := agent.participant.ReportSignal(
			signal.Kind,
			signal.GoalID,
			signal.TaskID,
			signal.Summary,
			signal.Detail,
		); err != nil {
			return err
		}
	}

	return nil
}

func (agent *Agent) observationSystem() (string, error) {
	template := viper.GetString(observationTemplatePath)
	if strings.TrimSpace(template) == "" {
		return "", errnie.Err(errnie.Validation, "observation prompt template is required", nil)
	}

	return agent.renderPrompt(template), nil
}

func (agent *Agent) renderPrompt(template string) string {
	rendered := strings.ReplaceAll(template, "{{ agent.role }}", agent.Role)
	rendered = strings.ReplaceAll(rendered, "{{ agent.name }}", agent.Name)
	rendered = strings.ReplaceAll(rendered, "{{ project.name }}", viper.GetString("project.name"))
	rendered = strings.ReplaceAll(rendered, "{{ project.description }}", viper.GetString("project.description"))
	rendered = strings.ReplaceAll(rendered, "{{ agent.characteristics }}", personaLinesFromViper(agent.Role, "characteristics"))
	rendered = strings.ReplaceAll(rendered, "{{ agent.responsibilities }}", personaLinesFromViper(agent.Role, "responsibilities"))
	rendered = strings.ReplaceAll(rendered, "{{ agent.guidelines }}", personaLinesFromViper(agent.Role, "guidelines"))

	return strings.TrimSpace(rendered)
}
