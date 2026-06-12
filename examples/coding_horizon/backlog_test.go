package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestBacklogNext verifies goal tasks precede hygiene tasks.
*/
func TestBacklogNext(t *testing.T) {
	Convey("Given a backlog with goal and hygiene tasks", t, func() {
		backlog := newBacklog("ship feature")

		backlog.Add(Task{ID: "hygiene-1", Kind: taskKindHygiene, Title: "split file"})
		backlog.Add(Task{ID: "goal-1", Kind: taskKindGoal, Title: "add retry"})
		backlog.Add(Task{ID: "hygiene-2", Kind: taskKindHygiene, Title: "add tests"})

		Convey("When Next is called", func() {
			first, ok := backlog.Next()

			Convey("Then it should prioritize the goal task", func() {
				So(ok, ShouldBeTrue)
				So(first.ID, ShouldEqual, "goal-1")
			})
		})
	})
}

/*
TestBacklogGoalSatisfied verifies goal completion detection.
*/
func TestBacklogGoalSatisfied(t *testing.T) {
	Convey("Given a backlog with one open goal task", t, func() {
		backlog := newBacklog("ship feature")
		backlog.Add(Task{ID: "goal-1", Kind: taskKindGoal, Title: "add retry"})

		Convey("When the goal task is still pending", func() {
			Convey("Then GoalSatisfied should be false", func() {
				So(backlog.GoalSatisfied(), ShouldBeFalse)
			})
		})

		Convey("When the goal task is done", func() {
			backlog.MarkDone("goal-1", "ok")

			Convey("Then GoalSatisfied should be true", func() {
				So(backlog.GoalSatisfied(), ShouldBeTrue)
			})
		})
	})
}

/*
TestHygieneTasksFromDigest verifies static debt extraction.
*/
func TestHygieneTasksFromDigest(t *testing.T) {
	Convey("Given a digest with lock signals", t, func() {
		digest := &RepoDigest{
			LockSignals: []string{"internal/bus.go contains \"sync.Mutex\""},
			SamplePaths: []string{"internal/bus.go"},
		}

		Convey("When hygieneTasksFromDigest runs", func() {
			tasks := hygieneTasksFromDigest(digest)

			Convey("Then it should emit a lock-free hygiene task", func() {
				So(len(tasks), ShouldBeGreaterThan, 0)
				So(tasks[0].Kind, ShouldEqual, taskKindHygiene)
			})
		})
	})
}
