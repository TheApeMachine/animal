package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

/*
TestAgentFineTuneExample verifies agent traces become fine-tuning examples.
*/
func TestAgentFineTuneExample(t *testing.T) {
	Convey("Given an agent trace and successful metric", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "user", Content: "prove it"}), ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "assistant", Content: "proof follows"}), ShouldBeNil)

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.GoalID = "goal-1"
		metric.Name = "goal_met"
		metric.Score = 1
		metric.Success = true

		Convey("When FineTuneExample is called", func() {
			example, err := agent.FineTuneExample(metric)

			Convey("Then it should include the system and trace messages", func() {
				So(err, ShouldBeNil)
				So(example.Messages, ShouldHaveLength, 3)
				So(example.Messages[0].Role, ShouldEqual, "system")
				So(example.Messages[1].Role, ShouldEqual, "user")
				So(example.Messages[2].Role, ShouldEqual, "assistant")
				So(example.Metadata["goal_id"], ShouldEqual, "goal-1")
			})
		})
	})
}

/*
TestAgentFineTuneExampleRejectsActorMismatch verifies traces cannot be recorded under another actor.
*/
func TestAgentFineTuneExampleRejectsActorMismatch(t *testing.T) {
	Convey("Given an agent and metric for a different actor", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		metric := swarm.NewMetricAt("other-actor", "Other", agent.Role, time.Now())
		metric.GoalID = "goal-1"
		metric.Name = "goal_met"
		metric.Score = 1
		metric.Success = true

		Convey("When FineTuneExample is called", func() {
			_, err := agent.FineTuneExample(metric)

			Convey("Then it should reject the metric", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestFineTuneExampleJSONL verifies JSONL encoding is newline terminated.
*/
func TestFineTuneExampleJSONL(t *testing.T) {
	Convey("Given a fine-tuning example", t, func() {
		example := FineTuneExample{
			Messages: []FineTuneMessage{
				{Role: "system", Content: "system"},
				{Role: "user", Content: "user"},
			},
		}

		Convey("When JSONL is called", func() {
			payload, err := example.JSONL()

			Convey("Then it should produce one JSONL line", func() {
				So(err, ShouldBeNil)
				So(string(payload), ShouldContainSubstring, `"messages"`)
				So(strings.HasSuffix(string(payload), "\n"), ShouldBeTrue)
			})
		})
	})
}

func BenchmarkAgentFineTuneExample(benchmark *testing.B) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		benchmark.Fatal(err)
	}

	if err := agent.Context.Append(provider.Message{Role: "user", Content: "prove it"}); err != nil {
		benchmark.Fatal(err)
	}

	metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
	metric.GoalID = "goal-1"
	metric.Name = "goal_met"
	metric.Score = 1
	metric.Success = true

	for benchmark.Loop() {
		if _, err := agent.FineTuneExample(metric); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkFineTuneExampleJSONL(benchmark *testing.B) {
	example := FineTuneExample{
		Messages: []FineTuneMessage{
			{Role: "system", Content: "system"},
			{Role: "user", Content: "user"},
		},
	}

	for benchmark.Loop() {
		if _, err := example.JSONL(); err != nil {
			benchmark.Fatal(err)
		}
	}
}
