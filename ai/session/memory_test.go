package session

import (
	"context"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	alcatraztool "github.com/theapemachine/animal/ai/tool/alcatraz"
	"github.com/theapemachine/qpool"
)

/*
TestCycleMemory verifies recall injection and post-generation consolidation.
*/
func TestCycleMemory(t *testing.T) {
	Convey("Given a session with local memory", t, func() {
		configureSessionTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		memory, err := ai.NewLocalMemory(ctx)
		So(err, ShouldBeNil)
		defer memory.Close()

		So(agent.UseMemory(memory), ShouldBeNil)
		So(agent.RememberMemory(ctx, ai.MemoryConsolidation{
			Records: []ai.MemoryRecord{
				{
					ID:         "goal-1:test-proof",
					Scope:      "goal-1",
					Text:       "Use make test output as proof.",
					Importance: 0.9,
				},
			},
		}), ShouldBeNil)

		terminal := newFakeTerminal("shell ready\n")
		bridge, err := alcatraztool.NewBridge(ctx, terminal)
		So(err, ShouldBeNil)

		streamer := &fakeStreamer{responses: [][]string{
			{`{"queries":[{"id":"","scope":"goal-1","text":"proof","limit":4,"vector_weight":0,"text_weight":1}]}`},
			{"make test\n"},
			{`{"records":[{"id":"goal-1:latest-proof","scope":"goal-1","text":"The assistant used make test output as proof.","importance":0.8}],"relationships":[],"forget":[]}`},
		}}

		session, err := NewSession(
			ctx,
			agent,
			streamer,
			bridge,
			provider.NewParams(),
		)
		So(err, ShouldBeNil)

		Convey("When Cycle is called", func() {
			result, cycleErr := session.Cycle()

			Convey("Then recall and consolidation should wrap the main generation", func() {
				So(cycleErr, ShouldBeNil)
				So(result.Status, ShouldEqual, StatusCompleted)
				So(streamer.schemas, ShouldResemble, []string{"memory_recall", "", "memory_consolidation"})
				So(streamer.contexts, ShouldHaveLength, 3)
				So(strings.Contains(streamer.contexts[1][0].Content, "Relevant memory:"), ShouldBeTrue)
				So(agent.Context.Messages, ShouldHaveLength, 2)
				So(strings.Contains(agent.Context.Messages[0].Content, "Relevant memory:"), ShouldBeFalse)

				packet, recallErr := agent.RecallMemory(ctx, ai.MemoryRecallPlan{
					Queries: []ai.MemoryQuery{
						{Scope: "goal-1", Text: "assistant", Limit: 4, TextWeight: 1},
					},
				})
				So(recallErr, ShouldBeNil)
				So(packet.Documents, ShouldHaveLength, 1)
				So(packet.Documents[0].ID, ShouldEqual, "goal-1:latest-proof")
			})
		})
	})
}

func BenchmarkCycleMemory(benchmark *testing.B) {
	configureSessionTestViper()
	ctx := context.Background()

	for benchmark.Loop() {
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		if err != nil {
			benchmark.Fatal(err)
		}

		memory, err := ai.NewLocalMemory(ctx)
		if err != nil {
			benchmark.Fatal(err)
		}

		if err := agent.UseMemory(memory); err != nil {
			benchmark.Fatal(err)
		}

		bridge, err := alcatraztool.NewBridge(ctx, newFakeTerminal("ready\n"))
		if err != nil {
			benchmark.Fatal(err)
		}

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{responses: [][]string{
				{`{"queries":[]}`},
				{"pwd\n"},
				{`{"records":[],"relationships":[],"forget":[]}`},
			}},
			bridge,
			provider.NewParams(),
		)
		if err != nil {
			benchmark.Fatal(err)
		}

		if _, err := session.Cycle(); err != nil {
			benchmark.Fatal(err)
		}

		_ = memory.Close()
	}
}
