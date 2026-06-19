package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/qpool"
)

func testSwarmOptions() Options {
	return Options{
		MeshID:    "test-swarm-mesh",
		GossipTTL: 30 * time.Second,
		MeshTTL:   time.Minute,
		Buffer:    8,
	}
}

/*
TestNewMesh verifies mesh construction and publish delivery.
*/
func TestNewMesh(t *testing.T) {
	Convey("Given a qpool and swarm options", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		mesh, err := NewMesh(ctx, pool, testSwarmOptions())
		So(err, ShouldBeNil)

		subscriber, err := mesh.Subscribe("actor-a", 8)
		So(err, ShouldBeNil)

		rumor := NewRumorAt(KindStatus, "actor-b", "Bob", "developer", time.Now())
		rumor.State = "idle"

		Convey("When actor-b publishes a rumor", func() {
			publishErr := mesh.Publish("actor-b", rumor)
			So(publishErr, ShouldBeNil)

			artifact, waitErr := waitBroadcastConsumer(ctx, subscriber, time.Second)
			So(waitErr, ShouldBeNil)

			payload := datura.As[Rumor](artifact)

			Convey("Then actor-a should receive the rumor", func() {
				So(payload.State, ShouldEqual, "idle")
			})
		})
	})
}

/*
TestNewRegistry verifies shared registry wiring.
*/
func TestNewRegistry(t *testing.T) {
	Convey("Given lease and swarm options", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})

		Convey("When NewParticipant is called twice", func() {
			first, firstErr := registry.NewParticipant("actor-a", "Ada", "developer", []string{"lanes/a/"})
			second, secondErr := registry.NewParticipant("actor-b", "Bob", "developer", []string{"lanes/b/"})

			Convey("Then both participants should share one mesh", func() {
				So(err, ShouldBeNil)
				So(firstErr, ShouldBeNil)
				So(secondErr, ShouldBeNil)
				So(registry.Mesh().GroupID(), ShouldEqual, testSwarmOptions().MeshID)
				So(first, ShouldNotBeNil)
				So(second, ShouldNotBeNil)
			})
		})
	})
}

/*
TestParticipantTryClaim verifies lease-backed claim publication.
*/
func TestParticipantTryClaim(t *testing.T) {
	Convey("Given two swarm participants", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		first, err := registry.NewParticipant("actor-a", "Ada", "developer", []string{"lanes/a/"})
		So(err, ShouldBeNil)

		second, err := registry.NewParticipant("actor-b", "Bob", "developer", nil)
		So(err, ShouldBeNil)

		Convey("When actor-a claims a prefix", func() {
			claimErr := first.TryClaim("lanes/a/")
			So(claimErr, ShouldBeNil)

			artifact, waitErr := waitParticipant(ctx, second, time.Second)
			So(waitErr, ShouldBeNil)

			rumor := datura.As[Rumor](artifact)
			So(rumor.Kind, ShouldEqual, KindClaim)

			Convey("And actor-b attempts the same prefix", func() {
				conflictErr := second.TryClaim("lanes/a/")

				Convey("Then the lease should reject the conflicting claim", func() {
					So(conflictErr, ShouldNotBeNil)
				})
			})
		})
	})
}

/*
TestParticipantPublishTaskSignalMetric verifies typed swarm artifact delivery.
*/
func TestParticipantPublishTaskSignalMetric(t *testing.T) {
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

		task := a2a.Task{
			ID: "task-1",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateSubmitted,
			},
			History: []a2a.Message{
				{
					Role: a2a.RoleUser,
					Parts: []a2a.Part{
						{Text: "Check the shared lease friction."},
					},
				},
			},
		}

		Convey("When task, signal, and metric artifacts are published", func() {
			So(first.PublishTask(task), ShouldBeNil)

			artifact, waitErr := waitParticipant(ctx, second, time.Second)
			So(waitErr, ShouldBeNil)
			So(second.ReceiveArtifact(artifact), ShouldBeNil)

			So(first.ReportSignal(
				SignalBlocker,
				"goal-1",
				"task-1",
				"lease unavailable",
				"prefix lanes/a is already claimed",
			), ShouldBeNil)

			artifact, waitErr = waitParticipant(ctx, second, time.Second)
			So(waitErr, ShouldBeNil)
			So(second.ReceiveArtifact(artifact), ShouldBeNil)

			So(first.ReportMetric(
				"goal-1",
				"task-1",
				"tests_passed",
				1,
				true,
				"go test ./...",
			), ShouldBeNil)

			artifact, waitErr = waitParticipant(ctx, second, time.Second)
			So(waitErr, ShouldBeNil)
			So(second.ReceiveArtifact(artifact), ShouldBeNil)

			Convey("Then the receiver view should merge all typed artifacts", func() {
				storedTask, ok := second.View().Task("task-1")
				So(ok, ShouldBeTrue)
				So(storedTask.Instruction(), ShouldEqual, "Check the shared lease friction.")
				So(len(second.View().RecentSignals()), ShouldEqual, 1)
				So(second.View().RecentSignals()[0].Kind, ShouldEqual, SignalBlocker)
				So(len(second.View().RecentMetrics()), ShouldEqual, 1)
				So(second.View().RecentMetrics()[0].Success, ShouldBeTrue)
			})
		})
	})
}
