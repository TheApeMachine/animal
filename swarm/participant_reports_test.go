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
TestParticipantReportSignal verifies report helpers update the local view.
*/
func TestParticipantReportSignal(t *testing.T) {
	Convey("Given a swarm participant", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		participant, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		Convey("When ReportSignal is called", func() {
			reportErr := participant.ReportSignal(
				SignalBlocker,
				"goal-1",
				"task-1",
				"lease blocked",
				"lanes/a is not available",
			)

			Convey("Then the local view should contain the blocker", func() {
				So(reportErr, ShouldBeNil)
				So(len(participant.View().BlockingSignals()), ShouldEqual, 1)
				So(participant.View().BlockingSignals()[0].Summary, ShouldEqual, "lease blocked")
			})
		})
	})
}

/*
TestParticipantReportMetric verifies metric helpers update the local view.
*/
func TestParticipantReportMetric(t *testing.T) {
	Convey("Given a swarm participant", t, func() {
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := NewRegistry(ctx, pool, testSwarmOptions(), lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		participant, err := registry.NewParticipant("actor-a", "Ada", "developer", nil)
		So(err, ShouldBeNil)

		Convey("When ReportMetric is called", func() {
			reportErr := participant.ReportMetric(
				"goal-1",
				"task-1",
				"tests_passed",
				1,
				true,
				"make test",
			)

			Convey("Then the local view should contain the metric", func() {
				So(reportErr, ShouldBeNil)
				So(len(participant.View().RecentMetrics()), ShouldEqual, 1)
				So(participant.View().RecentMetrics()[0].Name, ShouldEqual, "tests_passed")
			})
		})
	})
}
