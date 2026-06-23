package swarm

import (
	"time"

	"github.com/theapemachine/animal/a2a"
)

func dashboardView() (*View, error) {
	view, err := NewView(30 * time.Second)

	if err != nil {
		return nil, err
	}

	if err := seedDashboardView(view); err != nil {
		return nil, err
	}

	return view, nil
}

func seedDashboardView(view *View) error {
	if err := view.Merge(dashboardClaimRumor()); err != nil {
		return err
	}

	if err := view.Merge(dashboardStatusRumor()); err != nil {
		return err
	}

	if err := view.MergeTask(dashboardTask(a2a.TaskStateSubmitted)); err != nil {
		return err
	}

	if err := view.MergeTaskStatus(dashboardTaskStatus()); err != nil {
		return err
	}

	claim, err := dashboardTaskClaim()

	if err != nil {
		return err
	}

	if err := view.MergeTaskClaim(claim); err != nil {
		return err
	}

	if err := view.MergeSignal(dashboardSignal()); err != nil {
		return err
	}

	if err := view.MergeMetric(dashboardMetric()); err != nil {
		return err
	}

	contention, err := dashboardContention()

	if err != nil {
		return err
	}

	return view.MergeContention(contention)
}

func dashboardClaimRumor() Rumor {
	claim := NewRumorAt(KindClaim, "actor-a", "Ada", "developer", time.Now())
	claim.Prefix = "lanes/a/"

	return claim
}

func dashboardStatusRumor() Rumor {
	status := NewRumorAt(KindStatus, "actor-a", "Ada", "developer", time.Now())
	status.State = "working"

	return status
}

func dashboardTask(state a2a.TaskState) a2a.Task {
	return a2a.Task{
		ID: "task-1",
		Status: a2a.TaskStatus{
			State: state,
		},
	}
}

func dashboardTaskStatus() a2a.TaskStatusUpdateEvent {
	return a2a.TaskStatusUpdateEvent{
		TaskID: "task-1",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateWorking,
		},
	}
}

func dashboardTaskClaim() (TaskClaim, error) {
	return NewTaskClaimAt(
		"task-1",
		"actor-a",
		"Ada",
		"developer",
		time.Now(),
		time.Second,
	)
}

func dashboardSignal() Signal {
	signal := NewSignalAt(SignalFriction, "actor-a", "Ada", "developer", time.Now())
	signal.TaskID = "task-1"
	signal.Summary = "waiting on review"

	return signal
}

func dashboardMetric() Metric {
	metric := NewMetricAt("actor-a", "Ada", "developer", time.Now())
	metric.TaskID = "task-1"
	metric.Name = "tests_passed"
	metric.Score = 1
	metric.Success = true

	return metric
}

func dashboardContention() (Contention, error) {
	contention, err := NewContentionAt(
		"actor-b",
		"Bob",
		"developer",
		"lanes/a/",
		"lease conflict",
		time.Now(),
	)

	if err != nil {
		return Contention{}, err
	}

	contention.HolderID = "actor-a"
	contention.HolderPrefix = "lanes/a"

	return contention, nil
}
