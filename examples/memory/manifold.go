package main

import (
	"fmt"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/errnie"
)

/*
Manifold shows optional nomagique projection before DMT storage.
*/
func (demo *Demo) Manifold() error {
	baseMemory, err := ai.NewLocalMemory(demo.ctx)
	if err != nil {
		return err
	}

	projector, err := ai.NewManifoldProjector(demo.ctx, ai.ManifoldConfig{
		Architecture: []int{manifoldInputDim, manifoldHiddenDim, manifoldLatentDim},
		Batch:        manifoldBatch,
		Alpha:        manifoldAlpha,
	})
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

	text := "Projection records latent energy surprise for recall quality."
	projection, err := projector.Project(demo.ctx, text)
	if err != nil {
		return err
	}

	if err := memory.Remember(demo.ctx, ai.MemoryConsolidation{
		Records: []ai.MemoryRecord{
			{
				ID:         "projected-memory",
				Scope:      manifoldScope,
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
				Scope:      manifoldScope,
				Text:       "latent energy surprise",
				Limit:      manifoldRecallLimit,
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

	fmt.Fprintln(demo.output, "== Manifold-projected memory ==")
	fmt.Fprintf(demo.output, "projection_embedding_dim=%d\n", len(projection.Embedding))
	fmt.Fprintf(demo.output, "projection_energy=%.6f\n", projection.Energy)
	fmt.Fprintf(demo.output, "projection_surprise=%.6f\n", projection.Surprise)
	fmt.Fprintf(demo.output, "memory_embedding_dim=%d\n", len(packet.Documents[0].Embedding))
	fmt.Fprintln(demo.output, packet.Format())

	return nil
}
