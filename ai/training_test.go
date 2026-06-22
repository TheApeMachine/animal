package ai

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		defer func() {
			So(recorder.Close(), ShouldBeNil)
		}()

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.Name = "goal_met"
		metric.Score = 1
		metric.Success = true

		Convey("It should append one JSONL fine-tuning example", func() {
			err = recorder.Record(agent, metric)
			So(err, ShouldBeNil)

			var payload []byte
			payload, err = os.ReadFile(path)
			So(err, ShouldBeNil)

			lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
			So(len(lines), ShouldEqual, 1)

			var example FineTuneExample
			err = json.Unmarshal([]byte(lines[0]), &example)
			So(err, ShouldBeNil)
			So(len(example.Messages), ShouldBeGreaterThanOrEqualTo, 3)
			So(example.Messages[1].Role, ShouldEqual, "user")
			So(example.Messages[1].Content, ShouldEqual, "prove it")
			So(example.Messages[2].Role, ShouldEqual, "assistant")
			So(example.Messages[2].Content, ShouldEqual, "proof follows")
			So(example.Metadata["success"], ShouldEqual, "true")
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
		path := filepath.Join(t.TempDir(), "training.jsonl")
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		recorder, err := NewTrainingRecorder(ctx, path)
		So(err, ShouldBeNil)
		defer func() {
			So(recorder.Close(), ShouldBeNil)
		}()

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.Name = "goal_met"
		metric.Score = 0
		metric.Success = false

		Convey("It should reject the metric without writing JSONL", func() {
			err := recorder.Record(agent, metric)

			So(err, ShouldNotBeNil)
			_, statErr := os.Stat(path)
			So(os.IsNotExist(statErr), ShouldBeTrue)
		})
	})
}

func BenchmarkTrainingRecorderRecord(b *testing.B) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		b.Fatal(err)
	}

	if err := agent.Context.Append(provider.Message{Role: "user", Content: "prove it"}); err != nil {
		b.Fatal(err)
	}

	if err := agent.Context.Append(provider.Message{Role: "assistant", Content: "proof follows"}); err != nil {
		b.Fatal(err)
	}

	recorder, err := NewTrainingRecorder(ctx, filepath.Join(b.TempDir(), "training.jsonl"))
	if err != nil {
		b.Fatal(err)
	}
	defer recorder.Close()

	metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
	metric.Name = "goal_met"
	metric.Score = 1
	metric.Success = true

	for b.Loop() {
		if err := recorder.Record(agent, metric); err != nil {
			b.Fatal(err)
		}
	}
}
