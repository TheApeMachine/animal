package session

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	alcatraztool "github.com/theapemachine/animal/ai/tool/alcatraz"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

func testRegistry(
	ctx context.Context,
	pool *qpool.Q[any],
) (*swarm.Registry, error) {
	return swarm.NewRegistry(ctx, pool, swarm.Options{
		MeshID:    "session-task-test",
		GossipTTL: 30 * time.Second,
		MeshTTL:   time.Minute,
		Buffer:    8,
	}, lease.Options{
		KeySpace: lease.PathKeySpace{},
		IdleTTL:  time.Minute,
	})
}

func testTask() a2a.Task {
	return a2a.Task{
		ID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
		History: []a2a.Message{
			{
				Role: a2a.RoleUser,
				Parts: []a2a.Part{
					{Text: "Run make test."},
				},
			},
		},
		Metadata: map[string]any{
			metadataGoalID:      "goal-1",
			metadataLeasePrefix: "lanes/a/",
		},
	}
}

func TestRunTask(t *testing.T) {
	Convey("Given a swarm-attached session and A2A task", t, func() {
		configureSessionTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := testRegistry(ctx, pool)
		So(err, ShouldBeNil)

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/a/"})
		So(err, ShouldBeNil)

		terminal := newFakeTerminal("shell ready\n")
		bridge, err := alcatraztool.NewBridge(ctx, terminal)
		So(err, ShouldBeNil)

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{deltas: []string{"make test\n"}},
			bridge,
			provider.NewParams(),
		)
		So(err, ShouldBeNil)

		Convey("When RunTask is called", func() {
			result, runErr := session.RunTask(testTask())

			Convey("Then the task should complete and emit a success metric", func() {
				task, ok := agent.Participant().View().Task("task-1")
				So(runErr, ShouldBeNil)
				So(result.Status, ShouldEqual, StatusCompleted)
				So(terminal.writeBuffer.String(), ShouldEqual, "make test\n")
				So(ok, ShouldBeTrue)
				So(task.Status.State, ShouldEqual, a2a.TaskStateCompleted)
				So(len(agent.Participant().View().RecentMetrics()), ShouldEqual, 1)
				So(agent.Participant().View().RecentMetrics()[0].Name, ShouldEqual, "task_completed")
			})
		})
	})
}

func TestRunTaskLeaseBlocker(t *testing.T) {
	Convey("Given a task whose lease prefix is already claimed", t, func() {
		configureSessionTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := testRegistry(ctx, pool)
		So(err, ShouldBeNil)

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/a/"})
		So(err, ShouldBeNil)

		peer, err := registry.NewParticipant("peer", "Bob", "developer", []string{"lanes/a/"})
		So(err, ShouldBeNil)
		So(peer.TryClaim("lanes/a/"), ShouldBeNil)

		bridge, err := alcatraztool.NewBridge(ctx, newFakeTerminal("shell ready\n"))
		So(err, ShouldBeNil)

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{deltas: []string{"make test\n"}},
			bridge,
			provider.NewParams(),
		)
		So(err, ShouldBeNil)

		Convey("When RunTask is called", func() {
			result, runErr := session.RunTask(testTask())

			Convey("Then the task should fail and report a blocker", func() {
				task, ok := agent.Participant().View().Task("task-1")
				So(runErr, ShouldNotBeNil)
				So(result.Status, ShouldEqual, StatusFailed)
				So(ok, ShouldBeTrue)
				So(task.Status.State, ShouldEqual, a2a.TaskStateFailed)
				So(len(agent.Participant().View().BlockingSignals()), ShouldEqual, 1)
			})
		})
	})
}

func BenchmarkRunTask(benchmark *testing.B) {
	configureSessionTestViper()
	ctx := context.Background()

	for benchmark.Loop() {
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		registry, err := testRegistry(ctx, pool)
		if err != nil {
			benchmark.Fatal(err)
		}

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/a/"})
		if err != nil {
			benchmark.Fatal(err)
		}

		bridge, err := alcatraztool.NewBridge(ctx, newFakeTerminal("ready\n"))
		if err != nil {
			benchmark.Fatal(err)
		}

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{deltas: []string{"make test\n"}},
			bridge,
			provider.NewParams(),
		)
		if err != nil {
			benchmark.Fatal(err)
		}

		if _, err := session.RunTask(testTask()); err != nil {
			benchmark.Fatal(err)
		}
	}
}
