package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/storage"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

func configureAgentTestViper() {
	viper.Set("ai.prompt.template.system", "You are {{ agent.name }}, a {{ agent.role }}.")
	viper.Set("ai.prompt.template.observation", "You are {{ agent.name }}, the observation process for {{ project.name }}.")
	viper.Set("ai.prompt.template.memory_recall", "You are {{ agent.name }}, the memory recall process.")
	viper.Set("ai.prompt.template.memory_consolidation", "You are {{ agent.name }}, the memory consolidation process.")
	viper.Set("project.name", "Animal")
	viper.Set("project.description", "Multi-agent coordination harness.")
}

/*
TestAgentHandleIncoming verifies swarm rumors and provider messages in Cycle.
*/
func TestAgentHandleIncoming(t *testing.T) {
	Convey("Given a swarm-attached agent", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "agent-test-mesh",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    8,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		agent, err := NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/a/"})
		So(err, ShouldBeNil)

		peer, err := registry.NewParticipant("peer-b", "Bob", "developer", nil)
		So(err, ShouldBeNil)

		announce := swarm.NewRumorAt(swarm.KindAnnounce, "peer-b", "Bob", "developer", time.Now())
		announce.Topic = "roadmap.announce"
		announce.Payload = "ship leases first"

		Convey("When a peer publishes an announce rumor", func() {
			publishErr := peer.Announce(announce.Topic, announce.Payload)
			So(publishErr, ShouldBeNil)

			var artifact *datura.Artifact
			deadline := time.Now().Add(time.Second)

			for time.Now().Before(deadline) {
				artifact = agent.pollIncoming()
				if artifact != nil {
					break
				}

				time.Sleep(time.Millisecond)
			}

			if artifact == nil {
				t.Fatal("timed out waiting for announce rumor")
			}

			agent.handleIncoming(artifact)

			Convey("Then the agent view should contain the announcement", func() {
				records := agent.Participant().View().RecentAnnounces()
				So(len(records), ShouldEqual, 1)
				So(records[0].Topic, ShouldEqual, "roadmap.announce")
			})
		})
	})

	Convey("Given a legacy agent without swarm", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		message := provider.Message{ID: "m-1", Role: "user", Content: "hello"}

		Convey("When a provider message arrives", func() {
			artifact, artifactErr := qpool.NewBusArtifact(
				agent.ID,
				agent.ID,
				"message",
				message,
				time.Minute,
			)
			So(artifactErr, ShouldBeNil)

			agent.handleIncoming(artifact)

			Convey("Then it should append to agent context messages", func() {
				So(len(agent.Context.Messages), ShouldEqual, 1)
				So(agent.Context.Messages[0].Content, ShouldEqual, "hello")
			})
		})
	})
}

/*
TestAgentClone verifies clone-based sub-task delegation.
*/
func TestAgentClone(t *testing.T) {
	Convey("Given an agent with existing context", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "user", Content: "parent context"}), ShouldBeNil)

		task := a2a.Task{
			ID: "task-1",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateSubmitted,
			},
			History: []a2a.Message{
				{
					Role: a2a.RoleUser,
					Parts: []a2a.Part{
						{Text: "Investigate friction."},
					},
				},
			},
		}

		Convey("When CloneWithTask is called", func() {
			clone, cloneErr := agent.CloneWithTask(ctx, task)

			Convey("Then the clone should inherit context and append one sub-task message", func() {
				So(cloneErr, ShouldBeNil)
				So(clone.ID, ShouldNotEqual, agent.ID)
				So(clone.System, ShouldEqual, agent.System)
				So(len(agent.Context.Messages), ShouldEqual, 1)
				So(len(clone.Context.Messages), ShouldEqual, 2)
				So(clone.Context.Messages[0].Content, ShouldEqual, "parent context")
				So(clone.Context.Messages[1].Content, ShouldEqual, "Investigate friction.")
			})
		})
	})
}

/*
TestAgentUseTrainingRecorder verifies successful metric artifacts append training JSONL.
*/
func TestAgentUseTrainingRecorder(t *testing.T) {
	Convey("Given a swarm-attached agent with a training recorder", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "agent-training-test-mesh",
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
		So(agent.Context.Append(provider.Message{Role: "user", Content: "run tests"}), ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "assistant", Content: "tests pass"}), ShouldBeNil)

		path := filepath.Join(t.TempDir(), "training.jsonl")
		recorder, err := NewTrainingRecorder(ctx, path)
		So(err, ShouldBeNil)
		So(agent.UseTrainingRecorder(recorder), ShouldBeNil)

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.GoalID = "goal-1"
		metric.TaskID = "task-1"
		metric.Name = "tests_passed"
		metric.Score = 1
		metric.Success = true
		metric.Evidence = "make test"

		Convey("When a successful metric artifact arrives", func() {
			artifact, artifactErr := qpool.NewBusArtifact(
				agent.ID,
				agent.ID,
				swarm.MessageTypeMetric,
				metric,
				time.Minute,
			)
			So(artifactErr, ShouldBeNil)

			agent.handleIncoming(artifact)

			payload, readErr := os.ReadFile(path)

			Convey("Then one fine-tuning JSONL record should be written", func() {
				So(readErr, ShouldBeNil)
				So(string(payload), ShouldContainSubstring, `"messages"`)
				So(string(payload), ShouldContainSubstring, `"role":"system"`)
				So(string(payload), ShouldContainSubstring, `"metric":"tests_passed"`)
			})
		})
	})
}

/*
TestAgentUseTrainingStore verifies configured artifact training capture attaches to an agent.
*/
func TestAgentUseTrainingStore(t *testing.T) {
	Convey("Given a swarm-attached agent and storage config", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "agent-training-store-test-mesh",
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
		So(agent.Context.Append(provider.Message{Role: "user", Content: "run tests"}), ShouldBeNil)
		So(agent.Context.Append(provider.Message{Role: "assistant", Content: "tests pass"}), ShouldBeNil)

		trainingStore, err := agent.UseTrainingStore(ctx, storage.Config{
			Driver: storage.DriverBlob,
			Blob:   storage.BlobConfig{BucketURL: "mem://"},
		})
		So(err, ShouldBeNil)
		defer trainingStore.Close()

		metric := swarm.NewMetricAt(agent.ID, agent.Name, agent.Role, time.Now())
		metric.GoalID = "goal-1"
		metric.TaskID = "task-1"
		metric.Name = "tests_passed"
		metric.Score = 1
		metric.Success = true
		metric.Evidence = "make test"

		Convey("When a successful metric artifact arrives", func() {
			artifact, artifactErr := qpool.NewBusArtifact(
				agent.ID,
				agent.ID,
				swarm.MessageTypeMetric,
				metric,
				time.Minute,
			)
			So(artifactErr, ShouldBeNil)

			agent.handleIncoming(artifact)
			payload, exportErr := trainingStore.ExportGoal(ctx, "goal-1")

			Convey("Then one artifact-backed JSONL record should be exported", func() {
				So(exportErr, ShouldBeNil)
				So(string(payload), ShouldContainSubstring, `"messages"`)
				So(string(payload), ShouldContainSubstring, `"role":"assistant"`)
				So(string(payload), ShouldContainSubstring, `"metric":"tests_passed"`)
			})
		})
	})
}

/*
TestAgentUseTrainingStoreRejectsInvalidConfig verifies invalid storage is not attached.
*/
func TestAgentUseTrainingStoreRejectsInvalidConfig(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given an agent and invalid storage config", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		Convey("When UseTrainingStore is called", func() {
			trainingStore, err := agent.UseTrainingStore(ctx, storage.Config{})

			Convey("Then it should reject the config", func() {
				So(trainingStore, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(errnie.IsValidation(err), ShouldBeTrue)
			})
		})
	})
}

func BenchmarkAgentUseTrainingStore(benchmark *testing.B) {
	configureAgentTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
	if err != nil {
		benchmark.Fatal(err)
	}

	config := storage.Config{
		Driver: storage.DriverBlob,
		Blob:   storage.BlobConfig{BucketURL: "mem://"},
	}

	for benchmark.Loop() {
		trainingStore, err := agent.UseTrainingStore(ctx, config)

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = trainingStore.Close()
	}
}
