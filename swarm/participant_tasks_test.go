package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/qpool"
)

/*
TestParticipantSubmitTask verifies task submission broadcasts an A2A task.
*/
func TestParticipantSubmitTask(t *testing.T) {
	Convey("Given two swarm participants", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		first, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		second, err := registry.NewParticipant("actor-b", "Bob", "developer", nil)
		So(err, ShouldBeNil)

		Convey("When actor-a submits a task", func() {
			task, submitErr := first.SubmitTask(
				"task-1",
				"Investigate blocked lease.",
				map[string]any{"goal_id": "goal-1"},
			)
			So(submitErr, ShouldBeNil)
			So(task.Instruction(), ShouldEqual, "Investigate blocked lease.")

			artifact, waitErr := waitParticipant(ctx, second, time.Second)
			So(waitErr, ShouldBeNil)
			So(second.ReceiveArtifact(artifact), ShouldBeNil)

			Convey("Then the peer view should expose the submitted task", func() {
				submitted := second.View().SubmittedTasks()
				So(len(submitted), ShouldEqual, 1)
				So(submitted[0].ID, ShouldEqual, "task-1")
				So(submitted[0].Metadata["actor_id"], ShouldEqual, "actor-a")
			})
		})
	})
}

/*
TestParticipantStartTask verifies task status transitions are broadcast.
*/
func TestParticipantStartTask(t *testing.T) {
	Convey("Given two participants with a submitted task", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		first, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		second, err := registry.NewParticipant("actor-b", "Bob", "developer", nil)
		So(err, ShouldBeNil)

		_, submitErr := first.SubmitTask("task-1", "Inspect status.", nil)
		So(submitErr, ShouldBeNil)

		artifact, waitErr := waitParticipant(ctx, second, time.Second)
		So(waitErr, ShouldBeNil)
		So(second.ReceiveArtifact(artifact), ShouldBeNil)
		So(first.Drain(), ShouldBeNil)

		Convey("When actor-b starts the task", func() {
			startErr := second.StartTask("task-1", "working on it")
			So(startErr, ShouldBeNil)

			artifact, waitErr = waitParticipant(ctx, first, time.Second)
			So(waitErr, ShouldBeNil)
			So(first.ReceiveArtifact(artifact), ShouldBeNil)

			Convey("Then actor-a should see the task as working", func() {
				task, ok := first.View().Task("task-1")
				So(ok, ShouldBeTrue)
				So(task.Status.State, ShouldEqual, a2a.TaskStateWorking)
				So(task.Status.Message.Metadata["actor_id"], ShouldEqual, "actor-b")
			})
		})
	})
}

/*
TestParticipantCompleteTask verifies final completion events.
*/
func TestParticipantCompleteTask(t *testing.T) {
	Convey("Given a participant", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		participant, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		_, submitErr := participant.SubmitTask("task-1", "Verify completion.", nil)
		So(submitErr, ShouldBeNil)

		Convey("When CompleteTask is called", func() {
			completeErr := participant.CompleteTask("task-1", "verified")

			Convey("Then the local view should mark the task completed", func() {
				task, ok := participant.View().Task("task-1")
				So(completeErr, ShouldBeNil)
				So(ok, ShouldBeTrue)
				So(task.Status.State, ShouldEqual, a2a.TaskStateCompleted)
			})
		})
	})
}
