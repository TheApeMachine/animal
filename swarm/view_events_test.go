package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/a2a"
)

/*
TestViewMergeTaskStatus verifies A2A status events update local task state.
*/
func TestViewMergeTaskStatus(t *testing.T) {
	Convey("Given a view with an A2A task", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		task := a2a.Task{
			ID: "task-1",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateSubmitted,
			},
		}
		So(view.MergeTask(task), ShouldBeNil)

		event := a2a.TaskStatusUpdateEvent{
			TaskID: "task-1",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateWorking,
			},
		}

		Convey("When the status event is merged", func() {
			mergeErr := view.MergeTaskStatus(event)

			Convey("Then the task status should update", func() {
				storedTask, ok := view.Task("task-1")
				So(mergeErr, ShouldBeNil)
				So(ok, ShouldBeTrue)
				So(storedTask.Status.State, ShouldEqual, a2a.TaskStateWorking)
			})
		})
	})
}

/*
TestViewTasks verifies stable task snapshots and submitted filtering.
*/
func TestViewTasks(t *testing.T) {
	Convey("Given a view with submitted and working tasks", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		submitted := a2a.Task{
			ID: "task-b",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateSubmitted,
			},
			Metadata: map[string]any{"priority": "high"},
		}
		working := a2a.Task{
			ID: "task-a",
			Status: a2a.TaskStatus{
				State: a2a.TaskStateWorking,
			},
		}

		So(view.MergeTask(submitted), ShouldBeNil)
		So(view.MergeTask(working), ShouldBeNil)

		Convey("When Tasks and SubmittedTasks are called", func() {
			tasks := view.Tasks()
			submittedTasks := view.SubmittedTasks()
			tasks[1].Metadata["priority"] = "low"

			Convey("Then tasks should be stable and returned as clones", func() {
				storedTask, ok := view.Task("task-b")
				So(ok, ShouldBeTrue)
				So(tasks[0].ID, ShouldEqual, "task-a")
				So(tasks[1].ID, ShouldEqual, "task-b")
				So(len(submittedTasks), ShouldEqual, 1)
				So(submittedTasks[0].ID, ShouldEqual, "task-b")
				So(storedTask.Metadata["priority"], ShouldEqual, "high")
			})
		})
	})
}

/*
TestViewBlockingSignals verifies blocker filtering.
*/
func TestViewBlockingSignals(t *testing.T) {
	Convey("Given a view with mixed signals", t, func() {
		view, err := NewView(30 * time.Second)
		So(err, ShouldBeNil)

		blocker := NewSignalAt(SignalBlocker, "actor-a", "Ada", "developer", time.Now())
		blocker.Summary = "lease blocked"
		So(view.MergeSignal(blocker), ShouldBeNil)

		opportunity := NewSignalAt(SignalOpportunity, "actor-b", "Bob", "developer", time.Now())
		opportunity.Summary = "open task"
		So(view.MergeSignal(opportunity), ShouldBeNil)

		Convey("When BlockingSignals is called", func() {
			signals := view.BlockingSignals()

			Convey("Then only blockers should be returned", func() {
				So(len(signals), ShouldEqual, 1)
				So(signals[0].Kind, ShouldEqual, SignalBlocker)
			})
		})
	})
}

func BenchmarkViewMergeTask(b *testing.B) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		b.Fatal(err)
	}

	task := a2a.Task{
		ID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
	}

	for b.Loop() {
		if err := view.MergeTask(task); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkViewMergeTaskStatus(b *testing.B) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		b.Fatal(err)
	}

	task := a2a.Task{
		ID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
	}

	if err := view.MergeTask(task); err != nil {
		b.Fatal(err)
	}

	event := a2a.TaskStatusUpdateEvent{
		TaskID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateWorking,
		},
	}

	for b.Loop() {
		if err := view.MergeTaskStatus(event); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkViewMergeSignal(b *testing.B) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		b.Fatal(err)
	}

	signal := NewSignalAt(SignalBlocker, "actor-a", "Ada", "developer", time.Now())
	signal.Summary = "lease blocked"

	for b.Loop() {
		if err := view.MergeSignal(signal); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkViewTasks(b *testing.B) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		b.Fatal(err)
	}

	task := a2a.Task{
		ID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
	}

	if err := view.MergeTask(task); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_ = view.Tasks()
	}
}

func BenchmarkViewSubmittedTasks(b *testing.B) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		b.Fatal(err)
	}

	task := a2a.Task{
		ID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateSubmitted,
		},
	}

	if err := view.MergeTask(task); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_ = view.SubmittedTasks()
	}
}

func BenchmarkViewBlockingSignals(b *testing.B) {
	view, err := NewView(30 * time.Second)
	if err != nil {
		b.Fatal(err)
	}

	signal := NewSignalAt(SignalBlocker, "actor-a", "Ada", "developer", time.Now())
	signal.Summary = "lease blocked"

	if err := view.MergeSignal(signal); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		_ = view.BlockingSignals()
	}
}
