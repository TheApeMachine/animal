package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

/*
TestTrainingRecorderRecord verifies successful metrics are serialized as JSONL.
*/
func TestTrainingRecorderRecord(t *testing.T) {
	Convey("Given a training recorder and successful agent metric", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "training.jsonl")
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "user", Content: "prove it"}), ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "assistant", Content: "proof follows"}), ShouldBeNil)

		recorder, err := NewTrainingRecorder(ctx, path)
		So(err, ShouldBeNil)

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.Name = "goal_met"
		metric.Score = 1
		metric.Success = true

		Convey("When Record is called", func() {
			recordErr := recorder.Record(agent, metric)
			payload, readErr := os.ReadFile(path)

			Convey("Then it should append one JSONL fine-tuning example", func() {
				So(recordErr, ShouldBeNil)
				So(readErr, ShouldBeNil)
				So(string(payload), ShouldContainSubstring, `"role":"user"`)
				So(string(payload), ShouldContainSubstring, `"role":"assistant"`)
				So(string(payload), ShouldContainSubstring, `"success":"true"`)
			})
		})
	})
}

/*
TestTrainingRecorderRecordRejectsFailure verifies failed metrics are not collected.
*/
func TestTrainingRecorderRecordRejectsFailure(t *testing.T) {
	Convey("Given an unsuccessful metric", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		recorder, err := NewTrainingRecorder(ctx, filepath.Join(t.TempDir(), "training.jsonl"))
		So(err, ShouldBeNil)

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.Name = "goal_met"
		metric.Score = 0
		metric.Success = false

		Convey("When Record is called", func() {
			recordErr := recorder.Record(agent, metric)

			Convey("Then it should reject the metric", func() {
				So(recordErr, ShouldNotBeNil)
			})
		})
	})
}
