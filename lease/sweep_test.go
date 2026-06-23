package lease

import (
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestCoordinatorSweepExpired verifies expired leases are reclaimed with original prefixes.
*/
func TestCoordinatorSweepExpired(t *testing.T) {
	Convey("Given a coordinator with one idle lease and one active lease", t, func() {
		coordinator, err := NewCoordinator(Options{
			KeySpace: PathKeySpace{},
			IdleTTL:  20 * time.Millisecond,
		})
		So(err, ShouldBeNil)

		current := time.Unix(1000, 0)
		coordinator.now = func() time.Time {
			return current
		}

		So(coordinator.AcquireID("lanes/a/", "agent-a"), ShouldBeNil)
		current = current.Add(10 * time.Millisecond)
		So(coordinator.AcquireID("lanes/b/", "agent-b"), ShouldBeNil)
		current = current.Add(15 * time.Millisecond)

		Convey("When SweepExpired runs", func() {
			expired, err := coordinator.SweepExpired(current)

			Convey("Then only the idle lease should be returned and released", func() {
				So(err, ShouldBeNil)
				So(len(expired), ShouldEqual, 1)
				So(expired[0].ActorID, ShouldEqual, "agent-a")
				So(expired[0].Prefix, ShouldEqual, "lanes/a/")
				So(coordinator.AcquireID("lanes/a/", "agent-c"), ShouldBeNil)
				So(coordinator.AcquireID("lanes/b/", "agent-c"), ShouldNotBeNil)
			})
		})
	})
}

func BenchmarkCoordinatorSweepExpired(benchmark *testing.B) {
	benchmark.ReportAllocs()

	for benchmark.Loop() {
		coordinator, err := NewCoordinator(Options{
			KeySpace: PathKeySpace{},
			IdleTTL:  time.Nanosecond,
		})

		if err != nil {
			benchmark.Fatal(err)
		}

		for index := range 64 {
			err = coordinator.AcquireID(
				"lanes/bench/"+strconv.Itoa(index)+"/",
				"agent-a",
			)

			if err != nil {
				benchmark.Fatal(err)
			}
		}

		time.Sleep(time.Nanosecond)

		if _, err := coordinator.SweepExpired(time.Now()); err != nil {
			benchmark.Fatal(err)
		}
	}
}
