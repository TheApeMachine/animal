package main

import (
	"fmt"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/errnie"
)

/*
Friction shows DMT memory as a swarm hygiene backlog.
*/
func (demo *Demo) Friction() error {
	memory, err := ai.NewLocalMemory(demo.ctx)
	if err != nil {
		return err
	}
	defer memory.Close()

	if err := demo.seedFriction(memory); err != nil {
		return err
	}

	packet, err := memory.Recall(demo.ctx, ai.MemoryRecallPlan{
		Queries: []ai.MemoryQuery{
			{
				Scope:      frictionScope,
				Text:       "lease blocker prevents",
				Limit:      frictionRecallLimit,
				TextWeight: 1,
			},
			{
				Scope:      frictionScope,
				Text:       "quality drift appears",
				Limit:      frictionRecallLimit,
				TextWeight: 1,
			},
			{
				Scope:      frictionScope,
				Text:       "lease blocker quality drift",
				Limit:      frictionRecallLimit,
				TextWeight: 1,
			},
		},
	})
	if err != nil {
		return err
	}

	if len(packet.Documents) < 2 || len(packet.Relationships) == 0 {
		return errnie.Err(errnie.NotFound, "friction memory was not recalled", nil)
	}

	fmt.Fprintln(demo.output, "== DMT friction backlog ==")
	fmt.Fprintln(demo.output, packet.Format())

	return nil
}

func (demo *Demo) seedFriction(memory ai.Memory) error {
	return memory.Remember(demo.ctx, ai.MemoryConsolidation{
		Records: []ai.MemoryRecord{
			{
				ID:         "lease-blocker",
				Scope:      frictionScope,
				Text:       "Lease blocker prevents an agent from editing shared workspace files.",
				Importance: 0.9,
			},
			{
				ID:         "quality-drift",
				Scope:      frictionScope,
				Text:       "Quality drift appears when generated tests stop matching runtime behavior.",
				Importance: 0.8,
			},
			{
				ID:         "prompt-friction",
				Scope:      frictionScope,
				Text:       "Prompt friction appears when an agent only reports direct task changes.",
				Importance: 0.7,
			},
		},
		Relationships: []ai.MemoryRelationship{
			{
				ID:           "blocker-causes-drift",
				Scope:        frictionScope,
				FromID:       "lease-blocker",
				ToID:         "quality-drift",
				Relationship: "risks",
				Importance:   0.9,
			},
		},
	})
}
