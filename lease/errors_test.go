package lease

import (
	"errors"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestAsConflict verifies typed conflict error detection.
*/
func TestAsConflict(t *testing.T) {
	Convey("Given a ConflictError value", t, func() {
		conflict := &ConflictError{
			Key:      "lanes/a/sub",
			LeaseKey: "lanes/a",
			ActorID:  "builder-a",
		}

		Convey("When AsConflict is called", func() {
			parsed, ok := AsConflict(conflict)

			Convey("Then it should expose the holder", func() {
				So(ok, ShouldBeTrue)
				So(parsed.Key, ShouldEqual, conflict.Key)
				So(parsed.LeaseKey, ShouldEqual, conflict.LeaseKey)
				So(parsed.ActorID, ShouldEqual, conflict.ActorID)
			})
		})
	})

	Convey("Given an unrelated error", t, func() {
		Convey("When AsConflict is called", func() {
			_, ok := AsConflict(errors.New("other"))

			Convey("Then it should not match", func() {
				So(ok, ShouldBeFalse)
			})
		})
	})
}

/*
TestAsChanging verifies advisory error detection.
*/
func TestAsChanging(t *testing.T) {
	Convey("Given a ChangingError value", t, func() {
		changing := &ChangingError{
			Key:      "lanes/a/main.go",
			LeaseKey: "lanes/a",
			ActorID:  "builder-a",
		}

		Convey("When AsChanging is called", func() {
			parsed, ok := AsChanging(changing)

			Convey("Then it should recognize the advisory error", func() {
				So(ok, ShouldBeTrue)
				So(parsed.Key, ShouldEqual, changing.Key)
				So(parsed.LeaseKey, ShouldEqual, changing.LeaseKey)
				So(parsed.ActorID, ShouldEqual, changing.ActorID)
			})
		})
	})

	Convey("Given an unrelated error", t, func() {
		Convey("When AsChanging is called", func() {
			_, ok := AsChanging(errors.New("other"))

			Convey("Then it should not match", func() {
				So(ok, ShouldBeFalse)
			})
		})
	})
}
