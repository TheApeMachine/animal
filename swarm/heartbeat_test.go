package swarm

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/qpool"
)

/*
TestLeaseHeartbeatStart verifies holder-side renewal keeps an active lease alive.
*/
func TestLeaseHeartbeatStart(t *testing.T) {
	Convey("Given a participant holding a short-lived lease", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := heartbeatRegistry(ctx, pool, "heartbeat-renew-test")
		So(err, ShouldBeNil)

		holder, err := registry.NewParticipant("holder", "Ada", "developer", nil)
		So(err, ShouldBeNil)
		So(holder.TryClaim("lanes/a/"), ShouldBeNil)

		heartbeat, err := NewLeaseHeartbeat(ctx, holder, "lanes/a/", 10*time.Millisecond)
		So(err, ShouldBeNil)

		Convey("When the heartbeat runs past the coordinator idle TTL", func() {
			So(heartbeat.Start(), ShouldBeNil)
			time.Sleep(120 * time.Millisecond)

			claimant, err := registry.NewParticipant("claimant", "Ben", "developer", nil)
			So(err, ShouldBeNil)

			Convey("Then a peer still cannot acquire the renewed lease", func() {
				So(claimant.TryClaim("lanes/a/"), ShouldNotBeNil)
				So(heartbeat.Stop(), ShouldBeNil)
			})
		})
	})
}

/*
TestRegistrySweepExpiredLeases verifies missed heartbeats become release gossip.
*/
func TestRegistrySweepExpiredLeases(t *testing.T) {
	Convey("Given a peer view with a claimed prefix", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := heartbeatRegistry(ctx, pool, "heartbeat-sweep-test")
		So(err, ShouldBeNil)

		holder, err := registry.NewParticipant("holder", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		observer, err := registry.NewParticipant("observer", "Ona", "reviewer", nil)
		So(err, ShouldBeNil)

		So(holder.TryClaim("lanes/a/"), ShouldBeNil)
		So(receiveNext(ctx, observer), ShouldBeNil)
		So(observer.View().IsPrefixFree("lanes/a/"), ShouldBeFalse)

		Convey("When the registry sweeps after the idle TTL", func() {
			time.Sleep(80 * time.Millisecond)

			expired, err := registry.SweepExpiredLeases()
			So(err, ShouldBeNil)
			So(receiveNext(ctx, observer), ShouldBeNil)

			claimant, err := registry.NewParticipant("claimant", "Ben", "developer", nil)
			So(err, ShouldBeNil)

			Convey("Then the peer should see the prefix released and claimable", func() {
				So(expired, ShouldResemble, []string{"lanes/a/"})
				So(observer.View().IsPrefixFree("lanes/a/"), ShouldBeTrue)
				So(claimant.TryClaim("lanes/a/"), ShouldBeNil)
			})
		})
	})
}

/*
TestLeaseSweeperStart verifies the automatic sweeper emits release gossip.
*/
func TestLeaseSweeperStart(t *testing.T) {
	Convey("Given a running lease sweeper", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})
		defer pool.Close()

		registry, err := heartbeatRegistry(ctx, pool, "heartbeat-auto-test")
		So(err, ShouldBeNil)

		holder, err := registry.NewParticipant("holder", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		observer, err := registry.NewParticipant("observer", "Ona", "reviewer", nil)
		So(err, ShouldBeNil)

		So(holder.TryClaim("lanes/a/"), ShouldBeNil)
		So(receiveNext(ctx, observer), ShouldBeNil)

		sweeper, err := NewLeaseSweeper(ctx, registry, 10*time.Millisecond)
		So(err, ShouldBeNil)

		Convey("When the holder misses heartbeats", func() {
			So(sweeper.Start(), ShouldBeNil)
			err := waitPrefixFree(ctx, observer, "lanes/a/", 250*time.Millisecond)

			Convey("Then the sweeper should release the prefix over gossip", func() {
				So(err, ShouldBeNil)
				So(observer.View().IsPrefixFree("lanes/a/"), ShouldBeTrue)
				So(sweeper.Stop(), ShouldBeNil)
			})
		})
	})
}

func heartbeatRegistry(
	ctx context.Context,
	pool *qpool.Q[any],
	meshID string,
) (*Registry, error) {
	return NewRegistry(ctx, pool, Options{
		MeshID:    meshID,
		GossipTTL: time.Second,
		MeshTTL:   time.Second,
		Buffer:    16,
	}, lease.Options{
		KeySpace: lease.PathKeySpace{},
		IdleTTL:  50 * time.Millisecond,
	})
}

func receiveNext(ctx context.Context, participant *Participant) error {
	artifact, err := waitParticipant(ctx, participant, time.Second)

	if err != nil {
		return err
	}

	return participant.ReceiveArtifact(artifact)
}

func waitPrefixFree(
	ctx context.Context,
	participant *Participant,
	prefix string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := participant.Drain(); err != nil {
			return err
		}

		if participant.View().IsPrefixFree(prefix) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}

	return fmt.Errorf("swarm: prefix %q was not released", prefix)
}

func BenchmarkRegistrySweepExpiredLeases(benchmark *testing.B) {
	benchmark.ReportAllocs()

	for benchmark.Loop() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := heartbeatRegistry(ctx, pool, "heartbeat-benchmark")

		if err != nil {
			benchmark.Fatal(err)
		}

		holder, err := registry.NewParticipant("holder", "Ada", "developer", nil)

		if err != nil {
			benchmark.Fatal(err)
		}

		for index := range 16 {
			err = holder.TryClaim("lanes/bench/" + strconv.Itoa(index) + "/")

			if err != nil {
				benchmark.Fatal(err)
			}
		}

		time.Sleep(60 * time.Millisecond)

		if _, err := registry.SweepExpiredLeases(); err != nil {
			benchmark.Fatal(err)
		}

		pool.Close()
	}
}
