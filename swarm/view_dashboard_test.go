package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestViewClaims verifies active claim snapshots are deterministic.
*/
func TestViewClaims(t *testing.T) {
	Convey("Given a view with active claims", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		second := NewRumorAt(KindClaim, "actor-b", "Bob", "developer", time.Unix(2, 0))
		second.Prefix = "lanes/b/"
		first := NewRumorAt(KindClaim, "actor-a", "Ada", "developer", time.Unix(1, 0))
		first.Prefix = "lanes/a/"

		So(view.Merge(second), ShouldBeNil)
		So(view.Merge(first), ShouldBeNil)

		Convey("When Claims is called", func() {
			claims := view.Claims()

			Convey("Then claims should be ordered by prefix", func() {
				So(len(claims), ShouldEqual, 2)
				So(claims[0].Prefix, ShouldEqual, "lanes/a/")
				So(claims[0].ActorID, ShouldEqual, "actor-a")
				So(claims[1].Prefix, ShouldEqual, "lanes/b/")
			})
		})
	})
}

/*
TestViewStatuses verifies actor status snapshots are deterministic.
*/
func TestViewStatuses(t *testing.T) {
	Convey("Given a view with actor statuses", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		second := NewRumorAt(KindStatus, "actor-b", "Bob", "developer", time.Unix(2, 0))
		second.State = "waiting"
		first := NewRumorAt(KindStatus, "actor-a", "Ada", "developer", time.Unix(1, 0))
		first.State = "working"

		So(view.Merge(second), ShouldBeNil)
		So(view.Merge(first), ShouldBeNil)

		Convey("When Statuses is called", func() {
			statuses := view.Statuses()

			Convey("Then statuses should be ordered by actor", func() {
				So(len(statuses), ShouldEqual, 2)
				So(statuses[0].ActorID, ShouldEqual, "actor-a")
				So(statuses[0].State, ShouldEqual, "working")
				So(statuses[1].ActorID, ShouldEqual, "actor-b")
			})
		})
	})
}

/*
TestViewAllTaskClaims verifies all task claims are returned in stable order.
*/
func TestViewAllTaskClaims(t *testing.T) {
	Convey("Given a view with task claims across tasks", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		second, err := NewTaskClaimAt(
			"task-b",
			"actor-b",
			"Bob",
			"developer",
			time.Unix(2, 0),
			time.Second,
		)
		So(err, ShouldBeNil)

		first, err := NewTaskClaimAt(
			"task-a",
			"actor-a",
			"Ada",
			"developer",
			time.Unix(1, 0),
			time.Second,
		)
		So(err, ShouldBeNil)

		So(view.MergeTaskClaim(second), ShouldBeNil)
		So(view.MergeTaskClaim(first), ShouldBeNil)

		Convey("When AllTaskClaims is called", func() {
			claims := view.AllTaskClaims()

			Convey("Then task claims should be ordered by task then claim order", func() {
				So(len(claims), ShouldEqual, 2)
				So(claims[0].TaskID, ShouldEqual, "task-a")
				So(claims[0].ActorID, ShouldEqual, "actor-a")
				So(claims[1].TaskID, ShouldEqual, "task-b")
			})
		})
	})
}

func BenchmarkViewClaims(benchmark *testing.B) {
	view, err := NewView(30 * time.Second)

	if err != nil {
		benchmark.Fatal(err)
	}

	for index := range 64 {
		stamp := time.Unix(int64(index+1), 0)
		claim := NewRumorAt(
			KindClaim,
			"actor-a",
			"Ada",
			"developer",
			stamp,
		)
		claim.Prefix = "lanes/a/" + stamp.String()

		if err := view.Merge(claim); err != nil {
			benchmark.Fatal(err)
		}
	}

	benchmark.ReportAllocs()

	for benchmark.Loop() {
		_ = view.Claims()
	}
}
