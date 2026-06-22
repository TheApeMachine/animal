package ai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/spf13/viper"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/errnie"
)

const (
	memoryRecallTemplatePath        = "ai.prompt.template.memory_recall"
	memoryConsolidationTemplatePath = "ai.prompt.template.memory_consolidation"
)

/*
MemoryTurn is the temporary context used by recall and consolidation passes.
*/
type MemoryTurn struct {
	GoalID    string             `json:"goal_id"`
	TaskID    string             `json:"task_id"`
	Prompt    provider.Message   `json:"prompt"`
	Assistant provider.Message   `json:"assistant"`
	Memory    MemoryPacket       `json:"memory"`
	Messages  []provider.Message `json:"messages"`
}

/*
MemoryRecallStructuredOutput returns the schema used by the recall pass.
*/
func MemoryRecallStructuredOutput(strict bool) provider.StructuredOutput {
	return provider.StructuredOutput{
		Name:        "memory_recall",
		Description: "Queries for memories relevant to the next generation.",
		Strict:      strict,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"queries": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":            map[string]any{"type": "string"},
							"scope":         map[string]any{"type": "string"},
							"text":          map[string]any{"type": "string"},
							"limit":         map[string]any{"type": "integer", "minimum": 1, "maximum": 16},
							"vector_weight": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
							"text_weight":   map[string]any{"type": "number", "minimum": 0, "maximum": 1},
						},
						"required": []string{
							"id",
							"scope",
							"text",
							"limit",
							"vector_weight",
							"text_weight",
						},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"queries"},
			"additionalProperties": false,
		},
	}
}

/*
MemoryConsolidationStructuredOutput returns the schema used by the consolidation pass.
*/
func MemoryConsolidationStructuredOutput(strict bool) provider.StructuredOutput {
	return provider.StructuredOutput{
		Name:        "memory_consolidation",
		Description: "New memories, relationships, and forgotten IDs from the latest generation.",
		Strict:      strict,
		Schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"records": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":         map[string]any{"type": "string"},
							"scope":      map[string]any{"type": "string"},
							"text":       map[string]any{"type": "string"},
							"importance": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
						},
						"required":             []string{"id", "scope", "text", "importance"},
						"additionalProperties": false,
					},
				},
				"relationships": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":           map[string]any{"type": "string"},
							"scope":        map[string]any{"type": "string"},
							"from_id":      map[string]any{"type": "string"},
							"to_id":        map[string]any{"type": "string"},
							"relationship": map[string]any{"type": "string"},
							"importance":   map[string]any{"type": "number", "minimum": 0, "maximum": 1},
						},
						"required": []string{
							"id",
							"scope",
							"from_id",
							"to_id",
							"relationship",
							"importance",
						},
						"additionalProperties": false,
					},
				},
				"forget": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
			},
			"required":             []string{"records", "relationships", "forget"},
			"additionalProperties": false,
		},
	}
}

/*
MemoryRecallContext builds the temporary context for the recall pass.
*/
func (agent *Agent) MemoryRecallContext(
	ctx context.Context,
	turn MemoryTurn,
) (string, *provider.Context, error) {
	return agent.memoryContext(ctx, memoryRecallTemplatePath, turn)
}

/*
MemoryConsolidationContext builds the temporary context for the consolidation pass.
*/
func (agent *Agent) MemoryConsolidationContext(
	ctx context.Context,
	turn MemoryTurn,
) (string, *provider.Context, error) {
	return agent.memoryContext(ctx, memoryConsolidationTemplatePath, turn)
}

/*
ParseMemoryRecallPlan decodes recall structured output.
*/
func ParseMemoryRecallPlan(payload string) (MemoryRecallPlan, error) {
	var plan MemoryRecallPlan

	if strings.TrimSpace(payload) == "" {
		return MemoryRecallPlan{}, errnie.Err(errnie.Validation, "memory recall payload is required", nil)
	}

	if err := json.Unmarshal([]byte(payload), &plan); err != nil {
		return MemoryRecallPlan{}, errnie.Err(errnie.Validation, "memory recall decode failed", err)
	}

	if err := plan.Validate(); err != nil {
		return MemoryRecallPlan{}, err
	}

	return plan, nil
}

/*
ParseMemoryConsolidation decodes consolidation structured output.
*/
func ParseMemoryConsolidation(payload string) (MemoryConsolidation, error) {
	var consolidation MemoryConsolidation

	if strings.TrimSpace(payload) == "" {
		return MemoryConsolidation{}, errnie.Err(errnie.Validation, "memory consolidation payload is required", nil)
	}

	if err := json.Unmarshal([]byte(payload), &consolidation); err != nil {
		return MemoryConsolidation{}, errnie.Err(errnie.Validation, "memory consolidation decode failed", err)
	}

	if err := consolidation.Validate(); err != nil {
		return MemoryConsolidation{}, err
	}

	return consolidation, nil
}

/*
UseMemory attaches runtime memory to the agent.
*/
func (agent *Agent) UseMemory(memory Memory) error {
	if memory == nil {
		return errnie.Err(errnie.Validation, "agent memory is required", nil)
	}

	agent.Memory = append(agent.Memory, memory)

	return nil
}

/*
HasMemory reports whether the agent has runtime memory attached.
*/
func (agent *Agent) HasMemory() bool {
	return len(agent.Memory) > 0
}

/*
RecallMemory aggregates memory hits across attached memory stores.
*/
func (agent *Agent) RecallMemory(
	ctx context.Context,
	plan MemoryRecallPlan,
) (MemoryPacket, error) {
	if err := plan.Validate(); err != nil {
		return MemoryPacket{}, err
	}

	packet := MemoryPacket{
		Documents:     make([]MemoryDocument, 0),
		Relationships: make([]MemoryRelationship, 0),
	}

	for _, memory := range agent.Memory {
		recalled, err := memory.Recall(ctx, plan)
		if err != nil {
			return MemoryPacket{}, err
		}

		packet.Documents = append(packet.Documents, recalled.Documents...)
		packet.Relationships = append(packet.Relationships, recalled.Relationships...)
	}

	return packet, nil
}

/*
RememberMemory writes consolidation output to attached memory stores.
*/
func (agent *Agent) RememberMemory(
	ctx context.Context,
	consolidation MemoryConsolidation,
) error {
	if err := consolidation.Validate(); err != nil {
		return err
	}

	for _, memory := range agent.Memory {
		if err := memory.Remember(ctx, consolidation); err != nil {
			return err
		}
	}

	return nil
}

func (agent *Agent) memoryContext(
	ctx context.Context,
	templatePath string,
	turn MemoryTurn,
) (string, *provider.Context, error) {
	if ctx == nil {
		return "", nil, errnie.Err(errnie.Validation, "memory context is required", nil)
	}

	system, err := agent.memorySystem(templatePath)
	if err != nil {
		return "", nil, err
	}

	payload, err := json.Marshal(turn)
	if err != nil {
		return "", nil, errnie.Err(errnie.Validation, "memory turn marshal failed", err)
	}

	agentCtx := provider.NewContext(ctx)
	if err := agentCtx.Append(provider.Message{Role: "user", Content: string(payload)}); err != nil {
		return "", nil, err
	}

	return system, agentCtx, nil
}

func (agent *Agent) memorySystem(templatePath string) (string, error) {
	template := viper.GetString(templatePath)
	if strings.TrimSpace(template) == "" {
		return "", errnie.Err(errnie.Validation, "memory prompt template is required", nil)
	}

	return agent.renderPrompt(template), nil
}
