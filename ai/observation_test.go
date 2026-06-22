package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
TestObservationStructuredOutput verifies the swarm observation schema.
*/
func TestObservationStructuredOutput(t *testing.T) {
	Convey("Given strict observation structured output", t, func() {
		structured := ObservationStructuredOutput(true)

		Convey("It should define the required signal array", func() {
			So(structured.Name, ShouldEqual, "swarm_observation")
			So(structured.Strict, ShouldBeTrue)
			So(structured.Schema["type"], ShouldEqual, "object")
			So(structured.Schema["required"], ShouldResemble, []string{"signals"})
		})
	})
}

/*
TestAgentObservationContext verifies observation uses a temporary prompt context.
*/
func TestAgentObservationContext(t *testing.T) {
	Convey("Given an agent with a task conversation", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "user", Content: "main task"}), ShouldBeNil)

		request := ObservationRequest{
			GoalID:      "goal-1",
			TaskID:      "task-1",
			Instruction: "Run make test.",
			Prompt:      provider.Message{Role: "user", Content: "shell ready"},
			Assistant:   provider.Message{Role: "assistant", Content: "make test"},
			Messages:    agent.Context.Messages,
		}

		Convey("When ObservationContext is called", func() {
			system, observationContext, err := agent.ObservationContext(ctx, request)

			Convey("Then it should build a temporary observation context", func() {
				So(err, ShouldBeNil)
				So(system, ShouldContainSubstring, "observation process")
				So(observationContext.Messages, ShouldHaveLength, 1)
				So(observationContext.Messages[0].Role, ShouldEqual, "user")
				So(observationContext.Messages[0].Content, ShouldContainSubstring, `"goal_id":"goal-1"`)
				So(agent.Context.Messages, ShouldHaveLength, 1)
				So(agent.Context.Messages[0].Content, ShouldEqual, "main task")
			})
		})
	})
}

/*
TestParseObservation verifies structured observation JSON validation.
*/
func TestParseObservation(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a valid observation payload", t, func() {
		payload := `{"signals":[{"kind":"quality","goal_id":"goal-1","task_id":"","summary":"coverage gap","detail":"No tests were cited."}]}`

		Convey("When ParseObservation is called", func() {
			observation, err := ParseObservation(payload)

			Convey("Then it should decode signals", func() {
				So(err, ShouldBeNil)
				So(observation.Signals, ShouldHaveLength, 1)
				So(observation.Signals[0].Kind, ShouldEqual, swarm.SignalQuality)
			})
		})
	})

	Convey("Given an invalid observation payload", t, func() {
		payload := `{"signals":[{"kind":"quality","goal_id":"goal-1","task_id":"","summary":"","detail":"No tests were cited."}]}`

		Convey("When ParseObservation is called", func() {
			_, err := ParseObservation(payload)

			Convey("Then it should reject the signal", func() {
				So(err, ShouldNotBeNil)
				So(errnie.IsValidation(err), ShouldBeTrue)
			})
		})
	})
}

/*
TestAgentPublishObservation verifies observation signals reach the swarm view.
*/
func TestAgentPublishObservation(t *testing.T) {
	Convey("Given a swarm-attached agent", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "observation-test",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    8,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		agent, err := NewAgent(ctx, pool, "developer", "Ada", registry, nil)
		So(err, ShouldBeNil)

		observation := Observation{Signals: []ObservationSignal{
			{
				Kind:    swarm.SignalOpportunity,
				GoalID:  "goal-1",
				TaskID:  "",
				Summary: "reuse existing fixture",
				Detail:  "A nearby fixture can remove duplicated setup.",
			},
		}}

		Convey("When PublishObservation is called", func() {
			err := agent.PublishObservation(observation)

			Convey("Then the participant view should contain the signal", func() {
				signals := agent.Participant().View().SignalsByKind(swarm.SignalOpportunity)
				So(err, ShouldBeNil)
				So(signals, ShouldHaveLength, 1)
				So(strings.Contains(signals[0].Detail, "fixture"), ShouldBeTrue)
			})
		})
	})
}

func BenchmarkObservationContext(benchmark *testing.B) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		benchmark.Fatal(err)
	}

	request := ObservationRequest{
		GoalID:      "goal-1",
		TaskID:      "task-1",
		Instruction: "Run make test.",
		Prompt:      provider.Message{Role: "user", Content: "shell ready"},
		Assistant:   provider.Message{Role: "assistant", Content: "make test"},
	}

	for benchmark.Loop() {
		if _, _, err := agent.ObservationContext(ctx, request); err != nil {
			benchmark.Fatal(err)
		}
	}
}
