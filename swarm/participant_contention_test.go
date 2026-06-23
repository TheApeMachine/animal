package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/qpool"
)

/*
TestParticipantTryClaimPublishesContention verifies rejected claims become mesh events.
*/
func TestParticipantTryClaimPublishesContention(t *testing.T) {
	Convey("Given a held lease and an observer", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		holder, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		claimant, err := registry.NewParticipant("actor-b", "Bob", "developer", nil)
		So(err, ShouldBeNil)

		observer, err := registry.NewParticipant("actor-c", "Cy", "reviewer", nil)
		So(err, ShouldBeNil)

		So(holder.TryClaim("lanes/a/"), ShouldBeNil)
		artifact, err := waitParticipant(ctx, observer, time.Second)
		So(err, ShouldBeNil)
		So(observer.ReceiveArtifact(artifact), ShouldBeNil)

		Convey("When another actor claims an overlapping prefix", func() {
			err := claimant.TryClaim("lanes/a/sub/")
			So(err, ShouldNotBeNil)

			artifact, err = waitParticipant(ctx, observer, time.Second)
			So(err, ShouldBeNil)
			So(observer.ReceiveArtifact(artifact), ShouldBeNil)

			Convey("Then contention should be visible in the observer view", func() {
				contentions := observer.View().RecentContentions()
				So(len(contentions), ShouldEqual, 1)
				So(contentions[0].ActorID, ShouldEqual, "actor-b")
				So(contentions[0].HolderID, ShouldEqual, "actor-a")
				So(contentions[0].HolderPrefix, ShouldEqual, "lanes/a")
			})
		})
	})
}
