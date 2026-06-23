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
TestParticipantTryClaimConfiguredWithBackoff verifies rejected claims wait before retrying.
*/
func TestParticipantTryClaimConfiguredWithBackoff(t *testing.T) {
	Convey("Given a configured claimant racing a held prefix without gossip", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		holder, err := registry.NewParticipant("holder", "Ada", "developer", nil)
		So(err, ShouldBeNil)
		So(holder.TryClaim("lanes/a/"), ShouldBeNil)

		claimant, err := registry.NewParticipant(
			"claimant",
			"Ben",
			"developer",
			[]string{"lanes/a/"},
		)
		So(err, ShouldBeNil)

		backoff, err := NewContentionBackoff(
			ctx,
			2,
			time.Millisecond,
			time.Millisecond,
			0,
		)
		So(err, ShouldBeNil)

		waits := 0
		backoff.sleep = func(ctx context.Context, delay time.Duration) error {
			waits++

			return holder.Release("lanes/a/")
		}

		Convey("When TryClaimConfiguredWithBackoff is called", func() {
			prefix, err := claimant.TryClaimConfiguredWithBackoff(backoff)

			Convey("Then it should wait after rejection and claim on retry", func() {
				So(err, ShouldBeNil)
				So(prefix, ShouldEqual, "lanes/a/")
				So(waits, ShouldEqual, 1)
			})
		})
	})
}

/*
TestParticipantTryClaimConfiguredWithBackoffSkipsGossipHeld verifies no delay when gossip says unavailable.
*/
func TestParticipantTryClaimConfiguredWithBackoffSkipsGossipHeld(t *testing.T) {
	Convey("Given a claimant whose view already shows the prefix as held", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		holder, err := registry.NewParticipant("holder", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		claimant, err := registry.NewParticipant(
			"claimant",
			"Ben",
			"developer",
			[]string{"lanes/a/"},
		)
		So(err, ShouldBeNil)

		So(holder.TryClaim("lanes/a/"), ShouldBeNil)
		So(receiveNext(ctx, claimant), ShouldBeNil)

		backoff, err := NewContentionBackoff(
			ctx,
			2,
			time.Millisecond,
			time.Millisecond,
			0,
		)
		So(err, ShouldBeNil)

		waits := 0
		backoff.sleep = func(ctx context.Context, delay time.Duration) error {
			waits++

			return nil
		}

		Convey("When TryClaimConfiguredWithBackoff is called", func() {
			_, err := claimant.TryClaimConfiguredWithBackoff(backoff)

			Convey("Then it should not back off without a rejected claim", func() {
				So(err, ShouldNotBeNil)
				So(waits, ShouldEqual, 0)
			})
		})
	})
}
