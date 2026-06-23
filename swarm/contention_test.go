package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestNewContentionAt verifies contention event construction.
*/
func TestNewContentionAt(t *testing.T) {
	Convey("Given a failed claim attempt", t, func() {
		Convey("When a contention event is created", func() {
			contention, err := NewContentionAt(
				"actor-a",
				"Ada",
				"developer",
				"lanes/a/",
				"lease held",
				time.Unix(100, 0),
			)

			Convey("Then it should capture the actor and prefix", func() {
				So(err, ShouldBeNil)
				So(contention.ActorID, ShouldEqual, "actor-a")
				So(contention.Prefix, ShouldEqual, "lanes/a/")
				So(contention.Error, ShouldEqual, "lease held")
			})
		})
	})
}

/*
TestContentionValidate verifies malformed contention events fail explicitly.
*/
func TestContentionValidate(t *testing.T) {
	Convey("Given a contention without an error", t, func() {
		contention := Contention{
			ActorID: "actor-a",
			Prefix:  "lanes/a/",
			At:      time.Unix(100, 0).UnixNano(),
		}

		Convey("When Validate is called", func() {
			err := contention.Validate()

			Convey("Then validation should fail", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
