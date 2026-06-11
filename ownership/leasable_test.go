package ownership

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/lease"
)

func testLeaseOptions() lease.Options {
	return lease.Options{
		KeySpace: lease.PathKeySpace{},
		IdleTTL:  15 * time.Minute,
	}
}

/*
TestNewResource verifies resource construction.
*/
func TestNewResource(t *testing.T) {
	Convey("Given a nil coordinator", t, func() {
		Convey("When NewResource is called", func() {
			resource, err := NewResource(nil, "lanes/a/")

			Convey("Then it should reject the missing coordinator", func() {
				So(resource, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an empty lease key", t, func() {
		coordinator, err := lease.NewCoordinator(testLeaseOptions())
		So(err, ShouldBeNil)

		Convey("When NewResource is called", func() {
			resource, createErr := NewResource(coordinator, "")

			Convey("Then it should reject the empty key", func() {
				So(resource, ShouldBeNil)
				So(createErr, ShouldNotBeNil)
			})
		})
	})
}

/*
TestResourceRelease verifies release without a holder.
*/
func TestResourceRelease(t *testing.T) {
	Convey("Given an unheld resource", t, func() {
		coordinator, err := lease.NewCoordinator(testLeaseOptions())
		So(err, ShouldBeNil)

		resource, err := NewResource(coordinator, "lanes/a/")
		So(err, ShouldBeNil)

		Convey("When Release is called", func() {
			releaseErr := resource.Release()

			Convey("Then it should report that the resource is not held", func() {
				So(releaseErr, ShouldNotBeNil)
			})
		})
	})
}
