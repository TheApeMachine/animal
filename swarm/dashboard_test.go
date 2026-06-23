package swarm

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
)

/*
TestNewDashboard verifies dashboard construction validates required state.
*/
func TestNewDashboard(t *testing.T) {
	Convey("Given a view and refresh interval", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		Convey("When a dashboard is created", func() {
			dashboard, err := NewDashboard(
				context.Background(),
				view,
				DashboardOptions{Refresh: time.Second},
			)

			Convey("Then the dashboard should be ready to serve", func() {
				So(err, ShouldBeNil)
				So(dashboard.Close(), ShouldBeNil)
			})
		})
	})
}

/*
TestDashboardSnapshot verifies the dashboard read model covers swarm state.
*/
func TestDashboardSnapshot(t *testing.T) {
	Convey("Given a view with gossip, task, and metric state", t, func() {
		view, err := dashboardView()
		So(err, ShouldBeNil)

		dashboard, err := NewDashboard(
			context.Background(),
			view,
			DashboardOptions{Refresh: time.Second},
		)
		So(err, ShouldBeNil)

		defer func() {
			So(dashboard.Close(), ShouldBeNil)
		}()

		Convey("When Snapshot is called", func() {
			snapshot := dashboard.Snapshot()

			Convey("Then it should include dashboard-visible state", func() {
				So(len(snapshot.Claims), ShouldEqual, 1)
				So(len(snapshot.Statuses), ShouldEqual, 1)
				So(len(snapshot.Tasks), ShouldEqual, 1)
				So(len(snapshot.TaskClaims), ShouldEqual, 1)
				So(len(snapshot.Signals), ShouldEqual, 1)
				So(len(snapshot.Metrics), ShouldEqual, 1)
				So(len(snapshot.Contentions), ShouldEqual, 1)
				So(snapshot.Tasks[0].Status.State, ShouldEqual, a2a.TaskStateWorking)
			})
		})
	})
}

func BenchmarkDashboardSnapshot(benchmark *testing.B) {
	view, err := dashboardView()

	if err != nil {
		benchmark.Fatal(err)
	}

	dashboard, err := NewDashboard(
		context.Background(),
		view,
		DashboardOptions{Refresh: time.Second},
	)

	if err != nil {
		benchmark.Fatal(err)
	}

	benchmark.Cleanup(func() {
		if err := dashboard.Close(); err != nil {
			benchmark.Fatal(err)
		}
	})

	benchmark.ReportAllocs()

	for benchmark.Loop() {
		_ = dashboard.Snapshot()
	}
}
