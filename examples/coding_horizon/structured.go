package main

import (
	"context"
	"encoding/json"

	"github.com/theapemachine/animal/ai/provider"
)

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

func decodeStructured(
	openai *provider.OpenAI,
	ctx context.Context,
	system string,
	user string,
	output provider.StructuredOutput,
	target any,
) error {
	payload, err := openai.CompleteStructured(ctx, system, user, output)
	if err != nil {
		return err
	}

	return json.Unmarshal(payload, target)
}
