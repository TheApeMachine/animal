package ai

import (
	"strings"

	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
)

func (workflow *Workflow) validate(
	definition config.WorkflowYAML,
	registry *swarm.Registry,
) error {
	if registry == nil {
		return errnie.Err(errnie.Validation, "workflow registry is required", nil)
	}

	if definition.BroadcastGroupTTLSeconds <= 0 {
		return errnie.Err(errnie.Validation, "workflow broadcast ttl is required", nil)
	}

	if len(definition.Steps) == 0 {
		return errnie.Err(errnie.Validation, "workflow steps are required", nil)
	}

	for _, step := range definition.Steps {
		if err := workflow.validateStep(definition, step); err != nil {
			return err
		}
	}

	return nil
}

func (workflow *Workflow) validateStep(
	definition config.WorkflowYAML,
	step config.WorkflowStepYAML,
) error {
	if strings.TrimSpace(step.ID) == "" {
		return errnie.Err(errnie.Validation, "workflow step id is required", nil)
	}

	if step.StopCondition != "all_success" {
		return errnie.Err(errnie.Validation, "workflow stop must be all_success", nil)
	}

	switch workflow.stepMode(step) {
	case "work":
		return workflow.validateSlots(definition, step)
	case "conversation":
		return workflow.validateConversation(step)
	default:
		return errnie.Err(errnie.Validation, "workflow step mode is unsupported", nil)
	}
}

func (workflow *Workflow) validateSlots(
	definition config.WorkflowYAML,
	step config.WorkflowStepYAML,
) error {
	if len(step.Slots) == 0 {
		return errnie.Err(errnie.Validation, "workflow step slots are required", nil)
	}

	for _, slot := range step.Slots {
		if err := workflow.validateSlot(definition, step, slot); err != nil {
			return err
		}
	}

	return nil
}

func (workflow *Workflow) validateSlot(
	definition config.WorkflowYAML,
	step config.WorkflowStepYAML,
	slot config.WorkflowSlotYAML,
) error {
	if strings.TrimSpace(slot.Persona) == "" {
		return errnie.Err(errnie.Validation, "workflow slot persona is required", nil)
	}

	if slot.Replicas <= 0 {
		return errnie.Err(errnie.Validation, "workflow slot replicas are required", nil)
	}

	if !definition.FileLeasing || !step.RequireFileLeases || slot.ReadOnlyObserver {
		return nil
	}

	if len(slot.LeasePrefixes) == 0 {
		return errnie.Err(errnie.Validation, "workflow slot lease prefixes are required", nil)
	}

	return nil
}

func (workflow *Workflow) validateConversation(
	step config.WorkflowStepYAML,
) error {
	if step.Conversation == nil {
		return errnie.Err(errnie.Validation, "workflow conversation config is required", nil)
	}

	if step.Conversation.Rounds <= 0 {
		return errnie.Err(errnie.Validation, "workflow conversation rounds are required", nil)
	}

	if len(step.Conversation.Speakers) == 0 {
		return errnie.Err(errnie.Validation, "workflow conversation speakers are required", nil)
	}

	for _, speaker := range step.Conversation.Speakers {
		if strings.TrimSpace(speaker.Persona) == "" {
			return errnie.Err(errnie.Validation, "workflow conversation persona is required", nil)
		}
	}

	return nil
}

func (workflow *Workflow) stepMode(step config.WorkflowStepYAML) string {
	if strings.TrimSpace(step.Mode) == "" {
		return "work"
	}

	return strings.TrimSpace(step.Mode)
}
