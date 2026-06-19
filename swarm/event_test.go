package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestSignalValidate verifies swarm signal validation.
*/
func TestSignalValidate(t *testing.T) {
	Convey("Given a blocker signal", t, func() {
		signal := NewSignalAt(SignalBlocker, "actor-a", "Ada", "developer", time.Now())
		signal.Summary = "lease is unavailable"

		Convey("When Validate is called", func() {
			err := signal.Validate()

			Convey("Then the signal should be accepted", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a signal without a summary", t, func() {
		signal := NewSignalAt(SignalQuality, "actor-a", "Ada", "developer", time.Now())

		Convey("When Validate is called", func() {
			err := signal.Validate()

			Convey("Then validation should reject it", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestMetricValidate verifies normalized metric validation.
*/
func TestMetricValidate(t *testing.T) {
	Convey("Given a successful normalized metric", t, func() {
		metric := NewMetricAt("actor-a", "Ada", "developer", time.Now())
		metric.Name = "tests_passed"
		metric.Score = 1
		metric.Success = true

		Convey("When Validate is called", func() {
			err := metric.Validate()

			Convey("Then the metric should be accepted", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a metric outside the normalized range", t, func() {
		metric := NewMetricAt("actor-a", "Ada", "developer", time.Now())
		metric.Name = "tests_passed"
		metric.Score = 2

		Convey("When Validate is called", func() {
			err := metric.Validate()

			Convey("Then validation should reject it", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
