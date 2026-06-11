package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config mirrors cmd/cfg/config.yml (root keys after load).
type Config struct {
	AI AISection `yaml:"ai"`
}

// AISection is the nested `ai` block from config.yml.
type AISection struct {
	Model     string                  `yaml:"model"`
	Endpoint  string                  `yaml:"endpoint"`
	APIKey    string                  `yaml:"apiKey"`
	Lease     LeaseSection            `yaml:"lease"`
	Swarm     SwarmSection            `yaml:"swarm"`
	Prompt    PromptSection           `yaml:"prompt,omitempty"`
	Personas  map[string]PersonaYAML  `yaml:"personas"`
	Workflows map[string]WorkflowYAML `yaml:"workflows"`
}

// LeaseSection configures exclusive resource leasing defaults.
type LeaseSection struct {
	IdleTTLSeconds int `yaml:"idle_ttl_seconds"`
}

type PromptSection struct {
	Template PromptTemplateYAML `yaml:"template,omitempty"`
}

type PromptTemplateYAML struct {
	System string `yaml:"system,omitempty"`
}

// PersonaYAML is a reusable agent blueprint referenced by workflows.
type PersonaYAML struct {
	Role             string   `yaml:"role"`
	Characteristics  []string `yaml:"characteristics,omitempty"`
	Responsibilities []string `yaml:"responsibilities,omitempty"`
	Guidelines       []string `yaml:"guidelines,omitempty"`
}

// WorkflowYAML is a declarative orchestration blueprint.
type WorkflowYAML struct {
	Description              string             `yaml:"description,omitempty"`
	BroadcastGroupTTLSeconds int                `yaml:"broadcast_group_ttl_seconds"`
	FileLeasing              bool               `yaml:"file_leasing"`
	Steps                    []WorkflowStepYAML `yaml:"steps"`
}

// WorkflowStepYAML is one synchronous phase executed in order inside a workflow.
type WorkflowStepYAML struct {
	ID                string                `yaml:"id"`
	Description       string                `yaml:"description,omitempty"`
	Mode              string                `yaml:"mode"` // work (default) | conversation
	ParallelAgents    bool                  `yaml:"parallel_agents"`
	RequireFileLeases bool                  `yaml:"require_file_leases"`
	StopCondition     string                `yaml:"stop"`
	Slots             []WorkflowSlotYAML    `yaml:"slots"`
	Conversation      *ConversationStepYAML `yaml:"conversation,omitempty"`
}

// ConversationStepYAML schedules round-robin dialogue between personas.
type ConversationStepYAML struct {
	Rounds               int                           `yaml:"rounds"`
	Seed                 string                        `yaml:"seed"`
	BroadcastTurnSummary bool                          `yaml:"broadcast_turn_summary"`
	AttachWorkspaceTools bool                          `yaml:"attach_workspace_tools"`
	Speakers             []ConversationSpeakerSlotYAML `yaml:"speakers"`
}

// ConversationSpeakerSlotYAML references a persona for one seat in order.
type ConversationSpeakerSlotYAML struct {
	Persona string `yaml:"persona"`
}

// WorkflowSlotYAML describes one replicated agent cohort inside a phase.
type WorkflowSlotYAML struct {
	Persona          string   `yaml:"persona"`
	Replicas         int      `yaml:"replicas,omitempty"`
	ReadOnlyObserver bool     `yaml:"read_only"`
	LeasePrefixes    []string `yaml:"lease_prefixes"`
}

// WorkflowNames returns declared workflow identifiers in stable sorted order for diagnostics/UI.
func (c *Config) WorkflowNames() []string {
	var names []string
	for key := range c.AI.Workflows {
		names = append(names, key)
	}

	sort.Strings(names)

	return names
}

// Workflow returns a workflow definition by name.
func (c *Config) Workflow(name string) (WorkflowYAML, error) {
	for key, def := range c.AI.Workflows {
		if key == name {
			return def, nil
		}
	}

	return WorkflowYAML{}, fmt.Errorf("workflow %q is not declared in config", name)
}

// ResolveWorkflow selects an identifier declared under ai.workflows.
func (c *Config) ResolveWorkflow(preferred string) (string, error) {
	key := strings.TrimSpace(preferred)
	if key != "" {
		if _, err := c.Workflow(key); err != nil {
			return "", err
		}

		return key, nil
	}

	names := c.WorkflowNames()
	if len(names) == 0 {
		return "", fmt.Errorf("no workflows declared in config")
	}

	return names[0], nil
}

// Load parses a YAML file into Config.
func Load(path string) (*Config, error) {
	exp := filepath.Clean(os.ExpandEnv(path))

	bytes, readErr := os.ReadFile(exp)
	if readErr != nil {
		return nil, fmt.Errorf("config: read %s: %w", exp, readErr)
	}

	var cfg Config

	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", exp, err)
	}

	return &cfg, nil
}
