package main

import (
	"fmt"

	"github.com/theapemachine/animal/ai"
)

/*
Cognitive shows DMT radix memory recall and relationship recall.
*/
func (demo *Demo) Cognitive() error {
	memory, err := ai.NewLocalMemory(demo.ctx)
	if err != nil {
		return err
	}
	defer memory.Close()

	if err := demo.seedCognitive(memory); err != nil {
		return err
	}

	packet, err := memory.Recall(demo.ctx, ai.MemoryRecallPlan{
		Queries: []ai.MemoryQuery{
			{
				Scope:      cognitiveScope,
				Text:       "episodic REM consolidation",
				Limit:      cognitiveRecallLimit,
				TextWeight: 1,
			},
		},
	})
	if err != nil {
		return err
	}

	relationshipPacket, err := memory.Recall(demo.ctx, ai.MemoryRecallPlan{
		Queries: []ai.MemoryQuery{
			{
				Scope:      cognitiveScope,
				Text:       "dmt memory",
				Limit:      cognitiveRecallLimit,
				TextWeight: 1,
			},
		},
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(demo.output, "== DMT cognitive memory ==")
	fmt.Fprintln(demo.output, packet.Format())
	fmt.Fprintln(demo.output)
	fmt.Fprintln(demo.output, "== Relationship recall ==")
	fmt.Fprintln(demo.output, relationshipPacket.Format())

	return nil
}

func (demo *Demo) seedCognitive(memory ai.Memory) error {
	return memory.Remember(demo.ctx, ai.MemoryConsolidation{
		Records: []ai.MemoryRecord{
			{
				ID:         "dmt-cognitive-memory",
				Scope:      cognitiveScope,
				Text:       "DMT memory commits durable notes through episodic buffers and REM consolidation.",
				Importance: 0.9,
			},
			{
				ID:         "manifold-projection",
				Scope:      cognitiveScope,
				Text:       "Manifold projection adds latent embeddings before storage.",
				Importance: 0.7,
			},
		},
		Relationships: []ai.MemoryRelationship{
			{
				ID:           "memory-composition",
				Scope:        cognitiveScope,
				FromID:       "dmt-memory",
				ToID:         "manifold-projection",
				Relationship: "complements",
				Importance:   0.8,
			},
		},
	})
}
