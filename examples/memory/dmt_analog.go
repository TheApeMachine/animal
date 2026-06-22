package main

import (
	"fmt"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/errnie"
)

/*
Analog shows structural analog recall for a related task shape.
*/
func (demo *Demo) Analog() error {
	memory, err := ai.NewLocalMemory(demo.ctx)
	if err != nil {
		return err
	}
	defer memory.Close()

	if err := demo.seedAnalog(memory); err != nil {
		return err
	}

	packet, err := memory.Recall(demo.ctx, ai.MemoryRecallPlan{
		Queries: []ai.MemoryQuery{
			{
				Scope:      analogScope,
				Text:       "lease conflict terminal session",
				Limit:      analogRecallLimit,
				TextWeight: 1,
			},
		},
	})
	if err != nil {
		return err
	}

	if len(packet.Documents) == 0 {
		return errnie.Err(errnie.NotFound, "analog memory was not recalled", nil)
	}

	fmt.Fprintln(demo.output, "== DMT structural analog ==")
	fmt.Fprintln(demo.output, packet.Format())

	return nil
}

func (demo *Demo) seedAnalog(memory ai.Memory) error {
	return memory.Remember(demo.ctx, ai.MemoryConsolidation{
		Records: []ai.MemoryRecord{
			{
				ID:         "workspace-lease-route",
				Scope:      analogScope,
				Text:       "Lease conflict workspace files require switching to an unleased task.",
				Importance: 0.9,
			},
			{
				ID:         "schema-conflict-route",
				Scope:      analogScope,
				Text:       "Schema conflict artifact writes require checking media type support first.",
				Importance: 0.7,
			},
		},
	})
}
