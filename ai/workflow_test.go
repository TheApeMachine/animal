package ai

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

/*
TestWorkflowRun verifies configured stages spawn agents, gate leases, and publish task state.
*/
func TestWorkflowRun(t *testing.T) {
	Convey("Given a workflow with parallel lease-gated build slots", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 4, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "workflow-run-test",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    32,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		observer, err := registry.NewParticipant("observer", "Observer", "reviewer", nil)
		So(err, ShouldBeNil)

		workflow, err := NewWorkflow(ctx, pool)
		So(err, ShouldBeNil)

		definition := config.WorkflowYAML{
			Description:              "parallel build test",
			BroadcastGroupTTLSeconds: 60,
			FileLeasing:              true,
			Steps: []config.WorkflowStepYAML{
				{
					ID:                "build",
					Description:       "Build exclusive lanes.",
					ParallelAgents:    true,
					RequireFileLeases: true,
					StopCondition:     "all_success",
					Slots: []config.WorkflowSlotYAML{
						{
							Persona:       "developer",
							Replicas:      1,
							LeasePrefixes: []string{"lanes/a/"},
						},
						{
							Persona:       "developer",
							Replicas:      1,
							LeasePrefixes: []string{"lanes/b/"},
						},
						{
							Persona:          "reviewer",
							Replicas:         1,
							ReadOnlyObserver: true,
						},
					},
				},
			},
		}

		Convey("When Run is called", func() {
			result, err := workflow.Run(definition, registry)
			So(observer.Drain(), ShouldBeNil)

			claimer, err := registry.NewParticipant(
				"post-run-claimant", "Post", "developer", nil,
			)
			So(err, ShouldBeNil)

			Convey("Then all configured agents should complete and release their leases", func() {
				So(err, ShouldBeNil)
				So(result.Status, ShouldEqual, WorkflowStatusCompleted)
				So(len(result.Steps), ShouldEqual, 1)
				So(result.Steps[0].Status, ShouldEqual, WorkflowStatusCompleted)
				So(len(result.Steps[0].Agents), ShouldEqual, 3)
				So(result.Steps[0].Agents[0].Claimed, ShouldResemble, []string{"lanes/a/"})
				So(result.Steps[0].Agents[1].Claimed, ShouldResemble, []string{"lanes/b/"})
				So(result.Steps[0].Agents[2].ReadOnly, ShouldBeTrue)
				So(claimer.TryClaim("lanes/a/"), ShouldBeNil)
			})

			Convey("Then peer gossip should expose completed A2A tasks", func() {
				tasks := observer.View().Tasks()
				So(len(tasks), ShouldEqual, 3)

				for _, task := range tasks {
					So(task.Status.State, ShouldEqual, a2a.TaskStateCompleted)
				}
			})
		})
	})

	Convey("Given a workflow whose first step cannot acquire a required lease", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 2, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "workflow-stop-test",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    32,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		blocker, err := registry.NewParticipant("blocker", "Blocker", "developer", nil)
		So(err, ShouldBeNil)
		So(blocker.TryClaim("lanes/a/"), ShouldBeNil)

		workflow, err := NewWorkflow(ctx, pool)
		So(err, ShouldBeNil)

		definition := config.WorkflowYAML{
			Description:              "stop condition test",
			BroadcastGroupTTLSeconds: 60,
			FileLeasing:              true,
			Steps: []config.WorkflowStepYAML{
				{
					ID:                "build",
					Description:       "Build blocked lane.",
					RequireFileLeases: true,
					StopCondition:     "all_success",
					Slots: []config.WorkflowSlotYAML{
						{
							Persona:       "developer",
							Replicas:      1,
							LeasePrefixes: []string{"lanes/a/"},
						},
					},
				},
				{
					ID:            "review",
					StopCondition: "all_success",
					Slots: []config.WorkflowSlotYAML{
						{Persona: "reviewer", Replicas: 1},
					},
				},
			},
		}

		Convey("When Run is called", func() {
			result, err := workflow.Run(definition, registry)

			Convey("Then all_success should stop before the second step", func() {
				So(err, ShouldNotBeNil)
				So(result.Status, ShouldEqual, WorkflowStatusFailed)
				So(len(result.Steps), ShouldEqual, 1)
				So(result.Steps[0].Status, ShouldEqual, WorkflowStatusFailed)
				So(len(result.Steps[0].Agents), ShouldEqual, 1)
				So(result.Steps[0].Agents[0].Status, ShouldEqual, WorkflowStatusFailed)
				So(result.Steps[0].Agents[0].Error, ShouldContainSubstring, "held by actor")
			})
		})
	})
}

func BenchmarkWorkflowRun(benchmark *testing.B) {
	configureAgentTestViper()
	benchmark.ReportAllocs()

	definition := config.WorkflowYAML{
		Description:              "workflow benchmark",
		BroadcastGroupTTLSeconds: 60,
		FileLeasing:              true,
		Steps: []config.WorkflowStepYAML{
			{
				ID:                "build",
				Description:       "Build one lane.",
				RequireFileLeases: true,
				StopCondition:     "all_success",
				Slots: []config.WorkflowSlotYAML{
					{
						Persona:       "developer",
						Replicas:      1,
						LeasePrefixes: []string{"lanes/bench/"},
					},
				},
			},
		},
	}

	for benchmark.Loop() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "workflow-benchmark",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    16,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})

		if err != nil {
			benchmark.Fatal(err)
		}

		workflow, err := NewWorkflow(ctx, pool)

		if err != nil {
			benchmark.Fatal(err)
		}

		_, err = workflow.Run(definition, registry)

		if err != nil {
			benchmark.Fatal(err)
		}

		pool.Close()
	}
}
