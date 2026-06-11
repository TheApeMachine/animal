package lease

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func newPathCoordinator() (*Coordinator, error) {
	return NewCoordinator(Options{
		KeySpace: PathKeySpace{},
		IdleTTL:  15 * time.Minute,
	})
}

/*
TestNewCoordinator verifies coordinator construction.
*/
func TestNewCoordinator(t *testing.T) {
	Convey("Given a nil key space", t, func() {
		Convey("When NewCoordinator is called", func() {
			coordinator, err := NewCoordinator(Options{IdleTTL: 15 * time.Minute})

			Convey("Then it should reject the missing key space", func() {
				So(coordinator, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "key space is required")
			})
		})
	})

	Convey("Given a zero idle TTL", t, func() {
		Convey("When NewCoordinator is called", func() {
			coordinator, err := NewCoordinator(Options{KeySpace: PathKeySpace{}})

			Convey("Then it should reject the missing idle TTL", func() {
				So(coordinator, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "idle TTL is required")
			})
		})
	})

	Convey("Given a path key space", t, func() {
		Convey("When NewCoordinator is called", func() {
			coordinator, err := NewCoordinator(Options{
				KeySpace: PathKeySpace{},
				IdleTTL:  15 * time.Minute,
			})

			Convey("Then it should return a coordinator", func() {
				So(err, ShouldBeNil)
				So(coordinator, ShouldNotBeNil)
			})
		})
	})
}

/*
TestCoordinatorAcquireID verifies exclusive prefix acquisition by actor ID.
*/
func TestCoordinatorAcquireID(t *testing.T) {
	Convey("Given a fresh coordinator", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)

		Convey("When agent-a acquires lanes/a/", func() {
			acquireErr := coordinator.AcquireID("lanes/a/", "agent-a")
			So(acquireErr, ShouldBeNil)

			Convey("And agent-b attempts the same prefix", func() {
				conflictErr := coordinator.AcquireID("lanes/a/", "agent-b")

				Convey("Then the conflicting acquire should fail", func() {
					So(conflictErr, ShouldNotBeNil)
				})
			})
		})
	})
}

/*
TestCoordinatorReleaseID verifies prefix release by the holding actor.
*/
func TestCoordinatorReleaseID(t *testing.T) {
	Convey("Given agent-a holds lanes/a/", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)
		So(coordinator.AcquireID("lanes/a/", "agent-a"), ShouldBeNil)

		Convey("When agent-a releases the prefix", func() {
			releaseErr := coordinator.ReleaseID("lanes/a/", "agent-a")

			Convey("Then the lease should be dropped", func() {
				So(releaseErr, ShouldBeNil)
			})
		})
	})
}

/*
TestCoordinatorCanWrite verifies write authorization policy.
*/
func TestCoordinatorCanWrite(t *testing.T) {
	Convey("Given a principal that requires an active lease", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)

		principal := Principal{
			ActorID:      "agent-a",
			RequireLease: true,
		}

		Convey("When the actor writes without holding a lease", func() {
			writeErr := coordinator.CanWrite("lanes/a/main.go", principal)

			Convey("Then the write should be rejected", func() {
				So(writeErr, ShouldNotBeNil)
			})
		})

		Convey("When the actor holds lanes/a/", func() {
			So(coordinator.AcquireID("lanes/a/", "agent-a"), ShouldBeNil)

			Convey("And writes lanes/a/main.go", func() {
				writeErr := coordinator.CanWrite("lanes/a/main.go", principal)

				Convey("Then the write should be allowed", func() {
					So(writeErr, ShouldBeNil)
				})
			})
		})
	})

	Convey("Given a read-only principal", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)

		principal := Principal{ActorID: "reviewer", ReadOnly: true}

		Convey("When CanWrite is called", func() {
			writeErr := coordinator.CanWrite("main.go", principal)

			Convey("Then the write should be rejected", func() {
				So(writeErr, ShouldNotBeNil)
				So(writeErr.Error(), ShouldContainSubstring, "read-only")
			})
		})
	})

	Convey("Given a principal scoped to lanes/b/", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)

		principal := Principal{
			ActorID:         "builder-b",
			AllowedPrefixes: []string{"lanes/b/"},
		}

		Convey("When the actor writes outside the allowed prefix", func() {
			outsideErr := coordinator.CanWrite("lanes/a/main.go", principal)

			Convey("Then the write should be rejected", func() {
				So(outsideErr, ShouldNotBeNil)
			})
		})

		Convey("When the actor writes inside the allowed prefix", func() {
			insideErr := coordinator.CanWrite("lanes/b/main.go", principal)

			Convey("Then the write should be allowed", func() {
				So(insideErr, ShouldBeNil)
			})
		})
	})
}

/*
TestCoordinatorObserveRead verifies advisory read observation.
*/
func TestCoordinatorObserveRead(t *testing.T) {
	Convey("Given an unleased file", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)

		principal := Principal{ActorID: "reviewer"}

		Convey("When ObserveRead is called", func() {
			readErr := coordinator.ObserveRead("shared/main.go", principal)

			Convey("Then it should allow the read", func() {
				So(readErr, ShouldBeNil)
			})
		})
	})

	Convey("Given builder-a holds lanes/a/", t, func() {
		coordinator, err := newPathCoordinator()
		So(err, ShouldBeNil)
		So(coordinator.AcquireID("lanes/a/", "builder-a"), ShouldBeNil)

		Convey("When the lease holder reads lanes/a/main.go", func() {
			holder := Principal{ActorID: "builder-a"}
			readErr := coordinator.ObserveRead("lanes/a/main.go", holder)

			Convey("Then it should allow the read", func() {
				So(readErr, ShouldBeNil)
			})
		})

		Convey("When another actor reads lanes/a/main.go", func() {
			other := Principal{ActorID: "builder-b"}
			readErr := coordinator.ObserveRead("lanes/a/main.go", other)

			Convey("Then it should return an advisory changing error", func() {
				changing, ok := AsChanging(readErr)
				So(ok, ShouldBeTrue)
				So(changing.Key, ShouldEqual, "lanes/a/main.go")
				So(changing.ActorID, ShouldEqual, "builder-a")
				So(changing.LeaseKey, ShouldEqual, "lanes/a")
			})
		})
	})
}

/*
TestCoordinatorAcquireIDExpiresAfterIdle verifies forgotten releases do not lock forever.
*/
func TestCoordinatorAcquireIDExpiresAfterIdle(t *testing.T) {
	Convey("Given a coordinator with a short idle TTL", t, func() {
		coordinator, err := NewCoordinator(Options{
			KeySpace: PathKeySpace{},
			IdleTTL:  20 * time.Millisecond,
		})
		So(err, ShouldBeNil)

		So(coordinator.AcquireID("lanes/a/", "agent-a"), ShouldBeNil)

		Convey("When the lease sits idle past the TTL", func() {
			time.Sleep(30 * time.Millisecond)

			Convey("Then another actor should acquire the prefix", func() {
				So(coordinator.AcquireID("lanes/a/", "agent-b"), ShouldBeNil)
			})
		})
	})
}

/*
TestCoordinatorTouchID verifies explicit lease renewal.
*/
func TestCoordinatorTouchID(t *testing.T) {
	Convey("Given a coordinator with a short idle TTL and an active lease", t, func() {
		coordinator, err := NewCoordinator(Options{
			KeySpace: PathKeySpace{},
			IdleTTL:  40 * time.Millisecond,
		})
		So(err, ShouldBeNil)
		So(coordinator.AcquireID("lanes/a/", "agent-a"), ShouldBeNil)

		Convey("When the holder renews before idle expiry", func() {
			time.Sleep(25 * time.Millisecond)
			So(coordinator.TouchID("lanes/a/", "agent-a"), ShouldBeNil)
			time.Sleep(25 * time.Millisecond)

			Convey("Then another actor still cannot take the lease", func() {
				So(coordinator.AcquireID("lanes/a/", "agent-b"), ShouldNotBeNil)
			})
		})
	})
}
