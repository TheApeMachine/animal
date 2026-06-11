package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestRumorValidate verifies rumor field validation by kind.
*/
func TestRumorValidate(t *testing.T) {
	Convey("Given an announce rumor without a topic", t, func() {
		rumor := NewRumorAt(KindAnnounce, "actor-a", "Ada", "developer", time.Now())

		Convey("When Validate is called", func() {
			err := rumor.Validate()

			Convey("Then it should reject the missing topic", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given a claim rumor with prefix", t, func() {
		rumor := NewRumorAt(KindClaim, "actor-a", "Ada", "developer", time.Now())
		rumor.Prefix = "lanes/a/"

		Convey("When Validate is called", func() {
			err := rumor.Validate()

			Convey("Then it should accept the rumor", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}
