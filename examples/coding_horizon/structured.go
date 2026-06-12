package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/theapemachine/animal/ai/provider"
)

/*
Structured asks the model for JSON matching schema without tool access.
*/
type Structured struct {
	client openaiapi.Client
	model  string
}

func newStructured(endpoint, apiKey, model string) *Structured {
	return &Structured{
		client: openaiapi.NewClient(
			option.WithBaseURL(strings.TrimRight(endpoint, "/")),
			option.WithAPIKey(apiKey),
		),
		model: model,
	}
}

func (structured *Structured) Decode(
	ctx context.Context,
	system string,
	user string,
	output provider.StructuredOutput,
	target any,
) error {
	if err := output.Validate(); err != nil {
		return err
	}

	format := responses.ResponseFormatTextConfigParamOfJSONSchema(output.Name, output.Schema)
	if output.Strict && format.OfJSONSchema != nil {
		format.OfJSONSchema.Strict = openaiapi.Bool(true)
	}

	request := responses.ResponseNewParams{
		Model: structured.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(user, responses.EasyInputMessageRoleUser),
			},
		},
		Text: responses.ResponseTextConfigParam{Format: format},
	}

	if strings.TrimSpace(system) != "" {
		request.Instructions = param.NewOpt(system)
	}

	response, err := structured.client.Responses.New(ctx, request)
	if err != nil {
		return fmt.Errorf("coding horizon: structured response: %w", err)
	}

	payload := strings.TrimSpace(response.OutputText())
	if payload == "" {
		return fmt.Errorf("coding horizon: structured response returned empty output")
	}

	if err := json.Unmarshal([]byte(payload), target); err != nil {
		return fmt.Errorf("coding horizon: decode structured output: %w", err)
	}

	return nil
}

type intakeResult struct {
	GoalTasks []Task `json:"goal_tasks"`
	Summary   string `json:"summary"`
}

type planSlice struct {
	Steps        []string `json:"steps"`
	PrimaryFile  string   `json:"primary_file"`
	OldFragment  string   `json:"old_fragment"`
	NewFragment  string   `json:"new_fragment"`
	StopReason   string   `json:"stop_reason"`
	EvidenceUsed []string `json:"evidence_used"`
}

type auditVerdict struct {
	GoalMet          bool     `json:"goal_met"`
	RemainingRisks   []string `json:"remaining_risks"`
	HygieneRemaining int      `json:"hygiene_remaining"`
	Continue         bool     `json:"continue"`
	Summary          string   `json:"summary"`
}

var intakeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"goal_tasks": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":           map[string]any{"type": "string"},
					"kind":         map[string]any{"type": "string"},
					"title":        map[string]any{"type": "string"},
					"rationale":    map[string]any{"type": "string"},
					"target_files": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"acceptance":   map[string]any{"type": "string"},
				},
				"required":             []string{"id", "kind", "title", "rationale", "target_files", "acceptance"},
				"additionalProperties": false,
			},
		},
		"summary": map[string]any{"type": "string"},
	},
	"required":             []string{"goal_tasks", "summary"},
	"additionalProperties": false,
}

var planSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"steps":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"primary_file":  map[string]any{"type": "string"},
		"old_fragment":  map[string]any{"type": "string"},
		"new_fragment":  map[string]any{"type": "string"},
		"stop_reason":   map[string]any{"type": "string"},
		"evidence_used": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required":             []string{"steps", "primary_file", "old_fragment", "new_fragment", "stop_reason", "evidence_used"},
	"additionalProperties": false,
}

var auditSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"goal_met":          map[string]any{"type": "boolean"},
		"remaining_risks":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"hygiene_remaining": map[string]any{"type": "integer"},
		"continue":          map[string]any{"type": "boolean"},
		"summary":           map[string]any{"type": "string"},
	},
	"required":             []string{"goal_met", "remaining_risks", "hygiene_remaining", "continue", "summary"},
	"additionalProperties": false,
}
