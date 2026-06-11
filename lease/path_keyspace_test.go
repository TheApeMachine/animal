package lease

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestPathKeySpaceCovers verifies path-prefix coverage.
*/
func TestPathKeySpaceCovers(t *testing.T) {
	keySpace := PathKeySpace{}

	Convey("Given a prefix inside the same lane", t, func() {
		Convey("When Covers is called", func() {
			covered := keySpace.Covers("lanes/a/", "lanes/a/main.go")

			Convey("Then it should report coverage", func() {
				So(covered, ShouldBeTrue)
			})
		})
	})

	Convey("Given a prefix in a different lane", t, func() {
		Convey("When Covers is called", func() {
			covered := keySpace.Covers("lanes/b/", "lanes/a/main.go")

			Convey("Then it should not report coverage", func() {
				So(covered, ShouldBeFalse)
			})
		})
	})
}

/*
TestPathKeySpaceNormalize verifies path normalization.
*/
func TestPathKeySpaceNormalize(t *testing.T) {
	keySpace := PathKeySpace{}

	Convey("Given a workspace-relative path", t, func() {
		Convey("When Normalize is called", func() {
			normalized, err := keySpace.Normalize("./lanes/a/main.go")

			Convey("Then it should return a clean path", func() {
				So(err, ShouldBeNil)
				So(normalized, ShouldEqual, "lanes/a/main.go")
			})
		})
	})
}
