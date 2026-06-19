package ai

import (
	"context"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/storage"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
TestTrainingStoreRecord verifies successful metrics are stored as JSONL artifacts.
*/
func TestTrainingStoreRecord(t *testing.T) {
	Convey("Given a training store and successful agent metric", t, func() {
		agent, metric, trainingStore := mustTrainingStoreAgent(t)
		defer trainingStore.Close()

		Convey("When Record is called", func() {
			err := trainingStore.Record(agent, metric)
			records := make([]storage.Record, 0)

			if err == nil {
				records, err = trainingStore.store.List(context.Background(), "training/goal-1/")
			}

			Convey("Then it should write one training artifact under the goal prefix", func() {
				So(err, ShouldBeNil)
				So(records, ShouldHaveLength, 1)

				segments := strings.Split(records[0].Key, "/")
				So(segments, ShouldHaveLength, 5)
				So(segments[0], ShouldEqual, "training")
				So(segments[1], ShouldEqual, "goal-1")
				So(segments[2], ShouldEqual, agent.ID)
				So(segments[3], ShouldNotBeEmpty)
				So(segments[4], ShouldEndWith, ".jsonl")
				So(records[0].Artifact.Type(), ShouldEqual, trainingArtifactTypeJSONL)

				role, err := records[0].Artifact.Role()
				So(err, ShouldBeNil)
				So(role, ShouldEqual, "training")

				scope, err := records[0].Artifact.Scope()
				So(err, ShouldBeNil)
				So(scope, ShouldEqual, "goal-1")

				origin, err := records[0].Artifact.Origin()
				So(err, ShouldBeNil)
				So(origin, ShouldEqual, agent.ID)

				payload, err := records[0].Artifact.DecryptPayloadError()
				So(err, ShouldBeNil)
				So(string(payload), ShouldContainSubstring, `"role":"assistant"`)
				So(strings.HasSuffix(string(payload), "\n"), ShouldBeTrue)
			})
		})
	})
}

/*
TestTrainingStoreRecordRejectsMissingGoal verifies storage paths require goals.
*/
func TestTrainingStoreRecordRejectsMissingGoal(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a successful metric without a goal", t, func() {
		agent, metric, trainingStore := mustTrainingStoreAgent(t)
		defer trainingStore.Close()
		metric.GoalID = ""

		Convey("When Record is called", func() {
			err := trainingStore.Record(agent, metric)

			Convey("Then it should reject the metric", func() {
				So(err, ShouldNotBeNil)
				So(errnie.IsValidation(err), ShouldBeTrue)
			})
		})
	})
}

/*
TestTrainingStoreExportGoal verifies stored artifacts export as JSONL.
*/
func TestTrainingStoreExportGoal(t *testing.T) {
	Convey("Given two stored training examples for one goal", t, func() {
		agent, metric, trainingStore := mustTrainingStoreAgent(t)
		defer trainingStore.Close()

		So(trainingStore.Record(agent, metric), ShouldBeNil)
		So(trainingStore.Record(agent, metric), ShouldBeNil)

		Convey("When ExportGoal is called", func() {
			payload, err := trainingStore.ExportGoal(context.Background(), "goal-1")

			Convey("Then it should concatenate valid JSONL records", func() {
				So(err, ShouldBeNil)
				So(strings.Count(string(payload), "\n"), ShouldEqual, 2)
				So(string(payload), ShouldContainSubstring, `"goal_id":"goal-1"`)
			})
		})
	})
}

/*
TestTrainingStoreClose verifies closing cancels the training store scope.
*/
func TestTrainingStoreClose(t *testing.T) {
	Convey("Given a training store", t, func() {
		_, _, trainingStore := mustTrainingStoreAgent(t)

		Convey("When Close is called", func() {
			err := trainingStore.Close()

			Convey("Then it should close the artifact store and cancel context", func() {
				So(err, ShouldBeNil)
				So(trainingStore.ctx.Err(), ShouldNotBeNil)
			})
		})
	})
}

/*
TestNewTrainingStoreFromConfig verifies configured artifact-backed training storage.
*/
func TestNewTrainingStoreFromConfig(t *testing.T) {
	Convey("Given a blob storage config", t, func() {
		config := storage.Config{
			Driver: storage.DriverBlob,
			Blob:   storage.BlobConfig{BucketURL: "mem://"},
		}

		Convey("When NewTrainingStoreFromConfig is called", func() {
			trainingStore, err := NewTrainingStoreFromConfig(context.Background(), config)

			Convey("Then it should create a training store", func() {
				So(err, ShouldBeNil)
				So(trainingStore, ShouldNotBeNil)
				So(trainingStore.store, ShouldHaveSameTypeAs, &storage.BlobStore{})
				So(trainingStore.Close(), ShouldBeNil)
			})
		})
	})
}

func BenchmarkTrainingStoreRecord(benchmark *testing.B) {
	agent, metric, trainingStore := mustBenchmarkTrainingStoreAgent(benchmark)
	defer trainingStore.Close()

	for benchmark.Loop() {
		if err := trainingStore.Record(agent, metric); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkNewTrainingStoreFromConfig(benchmark *testing.B) {
	config := storage.Config{
		Driver: storage.DriverBlob,
		Blob:   storage.BlobConfig{BucketURL: "mem://"},
	}

	for benchmark.Loop() {
		trainingStore, err := NewTrainingStoreFromConfig(context.Background(), config)

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = trainingStore.Close()
	}
}

func BenchmarkTrainingStoreExportGoal(benchmark *testing.B) {
	agent, metric, trainingStore := mustBenchmarkTrainingStoreAgent(benchmark)
	defer trainingStore.Close()

	for range 8 {
		if err := trainingStore.Record(agent, metric); err != nil {
			benchmark.Fatal(err)
		}
	}

	for benchmark.Loop() {
		if _, err := trainingStore.ExportGoal(context.Background(), "goal-1"); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkTrainingStoreClose(benchmark *testing.B) {
	for benchmark.Loop() {
		_, _, trainingStore := mustBenchmarkTrainingStoreAgent(benchmark)

		if err := trainingStore.Close(); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func mustTrainingStoreAgent(t *testing.T) (*Agent, swarm.Metric, *TrainingStore) {
	t.Helper()

	agent, metric, trainingStore, err := newTrainingStoreAgent()
	if err != nil {
		t.Fatal(err)
	}

	return agent, metric, trainingStore
}

func mustBenchmarkTrainingStoreAgent(
	benchmark *testing.B,
) (*Agent, swarm.Metric, *TrainingStore) {
	benchmark.Helper()

	agent, metric, trainingStore, err := newTrainingStoreAgent()
	if err != nil {
		benchmark.Fatal(err)
	}

	return agent, metric, trainingStore
}

func newTrainingStoreAgent() (*Agent, swarm.Metric, *TrainingStore, error) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		return nil, swarm.Metric{}, nil, err
	}

	if err := agent.Context.Append(provider.Message{Role: "user", Content: "prove it"}); err != nil {
		return nil, swarm.Metric{}, nil, err
	}

	if err := agent.Context.Append(provider.Message{Role: "assistant", Content: "proof follows"}); err != nil {
		return nil, swarm.Metric{}, nil, err
	}

	trainingStore, err := NewTrainingStoreFromConfig(ctx, storage.Config{
		Driver: storage.DriverBlob,
		Blob:   storage.BlobConfig{BucketURL: "mem://"},
	})
	if err != nil {
		return nil, swarm.Metric{}, nil, err
	}

	metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
	metric.GoalID = "goal-1"
	metric.TaskID = "task-1"
	metric.Name = "goal_met"
	metric.Score = 1
	metric.Success = true

	return agent, metric, trainingStore, nil
}
