package ai

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
TestMemoryStructuredOutput verifies recall and consolidation schemas.
*/
func TestMemoryStructuredOutput(t *testing.T) {
	Convey("Given strict memory schemas", t, func() {
		recall := MemoryRecallStructuredOutput(true)
		consolidation := MemoryConsolidationStructuredOutput(true)

		Convey("They should define strict structured outputs", func() {
			So(recall.Name, ShouldEqual, "memory_recall")
			So(recall.Strict, ShouldBeTrue)
			So(consolidation.Name, ShouldEqual, "memory_consolidation")
			So(consolidation.Strict, ShouldBeTrue)
		})
	})
}

/*
TestAgentMemoryContexts verifies temporary memory contexts.
*/
func TestAgentMemoryContexts(t *testing.T) {
	Convey("Given an agent and memory turn", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "user", Content: "main task"}), ShouldBeNil)

		turn := MemoryTurn{
			GoalID: "goal-1",
			TaskID: "task-1",
			Prompt: provider.Message{
				Role:    "user",
				Content: "shell ready",
			},
			Messages: agent.Context.Messages,
		}

		Convey("When memory contexts are built", func() {
			recallSystem, recallContext, recallErr := agent.MemoryRecallContext(ctx, turn)
			consolidationSystem, consolidationContext, consolidationErr := agent.MemoryConsolidationContext(ctx, turn)

			Convey("Then they should not mutate the main context", func() {
				So(recallErr, ShouldBeNil)
				So(consolidationErr, ShouldBeNil)
				So(recallSystem, ShouldContainSubstring, "memory recall")
				So(consolidationSystem, ShouldContainSubstring, "memory consolidation")
				So(recallContext.Messages, ShouldHaveLength, 1)
				So(consolidationContext.Messages, ShouldHaveLength, 1)
				So(agent.Context.Messages, ShouldHaveLength, 1)
			})
		})
	})
}

/*
TestParseMemoryRecallPlan verifies recall structured output parsing.
*/
func TestParseMemoryRecallPlan(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given valid recall JSON", t, func() {
		payload := `{"queries":[{"id":"","scope":"goal-1","text":"proof","limit":4,"vector_weight":0,"text_weight":1}]}`

		Convey("When ParseMemoryRecallPlan is called", func() {
			plan, err := ParseMemoryRecallPlan(payload)

			Convey("Then it should decode queries", func() {
				So(err, ShouldBeNil)
				So(plan.Queries, ShouldHaveLength, 1)
				So(plan.Queries[0].Scope, ShouldEqual, "goal-1")
			})
		})
	})

	Convey("Given invalid recall JSON", t, func() {
		payload := `{"queries":[{"id":"","scope":"goal-1","text":"","limit":0,"vector_weight":0,"text_weight":1}]}`

		Convey("When ParseMemoryRecallPlan is called", func() {
			_, err := ParseMemoryRecallPlan(payload)

			Convey("Then it should reject invalid queries", func() {
				So(err, ShouldNotBeNil)
				So(errnie.IsValidation(err), ShouldBeTrue)
			})
		})
	})
}

/*
TestParseMemoryConsolidation verifies consolidation structured output parsing.
*/
func TestParseMemoryConsolidation(t *testing.T) {
	Convey("Given valid consolidation JSON", t, func() {
		payload := `{"records":[{"id":"goal-1:test-proof","scope":"goal-1","text":"Use make test output as proof.","importance":0.9}],"relationships":[],"forget":[]}`

		Convey("When ParseMemoryConsolidation is called", func() {
			consolidation, err := ParseMemoryConsolidation(payload)

			Convey("Then it should decode records", func() {
				So(err, ShouldBeNil)
				So(consolidation.Records, ShouldHaveLength, 1)
			})
		})
	})
}

/*
TestAgentUseMemory verifies memory attachment and aggregate recall.
*/
func TestAgentUseMemory(t *testing.T) {
	Convey("Given an agent with local memory", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		memory, err := NewLocalMemory(ctx)
		So(err, ShouldBeNil)
		defer memory.Close()

		So(agent.UseMemory(memory), ShouldBeNil)
		So(agent.RememberMemory(ctx, MemoryConsolidation{
			Records: []MemoryRecord{
				{
					ID:         "goal-1:test-proof",
					Scope:      "goal-1",
					Text:       "Use make test output as proof.",
					Importance: 0.9,
				},
			},
		}), ShouldBeNil)

		Convey("When RecallMemory is called", func() {
			packet, err := agent.RecallMemory(ctx, MemoryRecallPlan{
				Queries: []MemoryQuery{
					{Scope: "goal-1", Text: "proof", Limit: 4, TextWeight: 1},
				},
			})

			Convey("Then it should return memory hits", func() {
				So(err, ShouldBeNil)
				So(agent.HasMemory(), ShouldBeTrue)
				So(packet.Documents, ShouldHaveLength, 1)
			})
		})
	})
}

func BenchmarkMemoryRecallContext(benchmark *testing.B) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		benchmark.Fatal(err)
	}

	turn := MemoryTurn{
		GoalID: "goal-1",
		TaskID: "task-1",
		Prompt: provider.Message{
			Role:    "user",
			Content: "shell ready",
		},
	}

	for benchmark.Loop() {
		if _, _, err := agent.MemoryRecallContext(ctx, turn); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkMemoryConsolidationContext(benchmark *testing.B) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		benchmark.Fatal(err)
	}

	turn := MemoryTurn{
		GoalID: "goal-1",
		TaskID: "task-1",
		Prompt: provider.Message{
			Role:    "user",
			Content: "shell ready",
		},
		Assistant: provider.Message{
			Role:    "assistant",
			Content: "make test",
		},
		Memory: MemoryPacket{
			Documents: []MemoryDocument{
				{ID: "goal-1:test-proof", Scope: "goal-1", Text: "Use make test output as proof."},
			},
		},
	}

	for benchmark.Loop() {
		if _, _, err := agent.MemoryConsolidationContext(ctx, turn); err != nil {
			benchmark.Fatal(err)
		}
	}
}
