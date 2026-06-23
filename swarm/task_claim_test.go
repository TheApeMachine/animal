package swarm

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/qpool"
)

/*
TestTaskClaimReady verifies confirmation-window timing.
*/
func TestTaskClaimReady(t *testing.T) {
	Convey("Given a task claim with a confirmation window", t, func() {
		startedAt := time.Unix(1000, 0)
		claim, err := NewTaskClaimAt(
			"task-1",
			"actor-a",
			"Ada",
			"developer",
			startedAt,
			20*time.Millisecond,
		)
		So(err, ShouldBeNil)

		Convey("When Ready is checked before and after the window", func() {
			early := claim.Ready(startedAt.Add(10 * time.Millisecond))
			ready := claim.Ready(startedAt.Add(20 * time.Millisecond))

			Convey("Then it should only become confirmable after the window", func() {
				So(early, ShouldBeFalse)
				So(ready, ShouldBeTrue)
			})
		})
	})
}

/*
TestParticipantClaimTask verifies task claims are idempotent for one actor.
*/
func TestParticipantClaimTask(t *testing.T) {
	Convey("Given a participant with a visible submitted task", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := taskClaimRegistry(ctx, pool, "task-claim-idempotent")
		So(err, ShouldBeNil)

		submitter, err := registry.NewParticipant("submitter", "Sam", "project_manager", nil)
		So(err, ShouldBeNil)

		claimant, err := registry.NewParticipant("claimant", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		_, err = submitter.SubmitTask("task-1", "Build the lane.", nil)
		So(err, ShouldBeNil)
		So(receiveNext(ctx, claimant), ShouldBeNil)

		Convey("When the same actor claims the task twice", func() {
			firstClaim, err := claimant.ClaimTask("task-1", 20*time.Millisecond)
			So(err, ShouldBeNil)

			secondClaim, err := claimant.ClaimTask("task-1", 20*time.Millisecond)

			Convey("Then the second claim should return the original claim", func() {
				So(err, ShouldBeNil)
				So(secondClaim.At, ShouldEqual, firstClaim.At)
				So(len(claimant.View().TaskClaims("task-1")), ShouldEqual, 1)
			})
		})
	})
}

/*
TestParticipantConfirmTaskClaim verifies claim-confirm racing chooses one worker.
*/
func TestParticipantConfirmTaskClaim(t *testing.T) {
	Convey("Given two participants racing for one submitted task", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := taskClaimRegistry(ctx, pool, "task-claim-race")
		So(err, ShouldBeNil)

		submitter, err := registry.NewParticipant("submitter", "Sam", "project_manager", nil)
		So(err, ShouldBeNil)

		first, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		second, err := registry.NewParticipant("actor-b", "Ben", "developer", nil)
		So(err, ShouldBeNil)

		_, err = submitter.SubmitTask("task-1", "Build the lane.", nil)
		So(err, ShouldBeNil)
		So(receiveNext(ctx, first), ShouldBeNil)
		So(receiveNext(ctx, second), ShouldBeNil)

		Convey("When both actors claim before confirming", func() {
			_, err := first.ClaimTask("task-1", 20*time.Millisecond)
			So(err, ShouldBeNil)

			time.Sleep(time.Millisecond)

			_, err = second.ClaimTask("task-1", 20*time.Millisecond)
			So(err, ShouldBeNil)
			So(waitTaskClaims(ctx, first, "task-1", 2, time.Second), ShouldBeNil)
			So(waitTaskClaims(ctx, second, "task-1", 2, time.Second), ShouldBeNil)
			time.Sleep(25 * time.Millisecond)

			err = first.ConfirmTaskClaim("task-1", "confirmed winner")
			So(err, ShouldBeNil)
			err = second.ConfirmTaskClaim("task-1", "confirmed loser")

			Convey("Then only the deterministic winner should start work", func() {
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "actor-a")

				task, ok := first.View().Task("task-1")
				So(ok, ShouldBeTrue)
				So(task.Status.State, ShouldEqual, a2a.TaskStateWorking)
				So(task.Status.Message.Metadata["actor_id"], ShouldEqual, "actor-a")
			})
		})
	})
}

func taskClaimRegistry(
	ctx context.Context,
	pool *qpool.Q[any],
	meshID string,
) (*Registry, error) {
	return NewRegistry(ctx, pool, Options{
		MeshID:    meshID,
		GossipTTL: time.Second,
		MeshTTL:   time.Second,
		Buffer:    32,
	}, lease.Options{
		KeySpace: lease.PathKeySpace{},
		IdleTTL:  time.Minute,
	})
}

func waitTaskClaims(
	ctx context.Context,
	participant *Participant,
	taskID string,
	count int,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := participant.Drain(); err != nil {
			return err
		}

		if len(participant.View().TaskClaims(taskID)) >= count {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}

	return fmt.Errorf("swarm: timed out waiting for task claims")
}

func BenchmarkViewMergeTaskClaim(benchmark *testing.B) {
	benchmark.ReportAllocs()

	for benchmark.Loop() {
		view, err := NewView(time.Second)

		if err != nil {
			benchmark.Fatal(err)
		}

		for index := range 64 {
			claim, err := NewTaskClaimAt(
				"task-1",
				fmt.Sprintf("actor-%d", index),
				"Agent",
				"developer",
				time.Unix(1000, int64(index)),
				time.Millisecond,
			)

			if err != nil {
				benchmark.Fatal(err)
			}

			if err := view.MergeTaskClaim(claim); err != nil {
				benchmark.Fatal(err)
			}
		}
	}
}
