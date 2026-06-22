package main

import (
	"fmt"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/errnie"
)

/*
Projection shows projection-aware memory with a local projector.
*/
func (demo *Demo) Projection() error {
	baseMemory, err := ai.NewLocalMemory(demo.ctx)
	if err != nil {
		return err
	}

	projector, err := NewSignalProjector(demo.ctx)
	if err != nil {
		baseMemory.Close()

		return err
	}

	memory, err := ai.NewProjectedMemory(demo.ctx, baseMemory, projector)
	if err != nil {
		projector.Close()
		baseMemory.Close()

		return err
	}
	defer memory.Close()

	text := "Structured outputs surface friction and quality signals before drift."
	projection, err := projector.Project(demo.ctx, text)
	if err != nil {
		return err
	}

	if err := memory.Remember(demo.ctx, ai.MemoryConsolidation{
		Records: []ai.MemoryRecord{
			{
				ID:         "signal-projected-memory",
				Scope:      projectionScope,
				Text:       text,
				Importance: 0.9,
			},
		},
	}); err != nil {
		return err
	}

	packet, err := memory.Recall(demo.ctx, ai.MemoryRecallPlan{
		Queries: []ai.MemoryQuery{
			{
				Scope:      projectionScope,
				Text:       "friction quality signals",
				Limit:      projectionRecallLimit,
				TextWeight: 1,
			},
		},
	})
	if err != nil {
		return err
	}

	if len(packet.Documents) == 0 {
		return errnie.Err(errnie.NotFound, "projected memory was not recalled", nil)
	}

	fmt.Fprintln(demo.output, "== Projected memory ==")
	fmt.Fprintf(demo.output, "projection_embedding_dim=%d\n", len(projection.Embedding))
	fmt.Fprintf(demo.output, "projection_energy=%.6f\n", projection.Energy)
	fmt.Fprintf(demo.output, "projection_surprise=%.6f\n", projection.Surprise)
	fmt.Fprintf(demo.output, "memory_embedding_dim=%d\n", len(packet.Documents[0].Embedding))
	fmt.Fprintln(demo.output, packet.Format())

	return nil
}
