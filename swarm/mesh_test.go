package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/lease"
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

			qv, waitErr := waitBroadcastConsumer(ctx, subscriber, time.Second)
			So(waitErr, ShouldBeNil)

			payload, ok := qv.Value.(Rumor)

			Convey("Then actor-a should receive the rumor", func() {
				So(ok, ShouldBeTrue)
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

			qv, waitErr := waitParticipant(ctx, second, time.Second)
			So(waitErr, ShouldBeNil)

			rumor, ok := qv.Value.(Rumor)
			So(ok, ShouldBeTrue)
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
