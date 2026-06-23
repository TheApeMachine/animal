package swarm

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/errnie"
)

/*
MergeTask applies an A2A task into the local view.
*/
func (view *View) MergeTask(task a2a.Task) error {
	if err := task.Validate(); err != nil {
		return err
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)
		updated.tasks[task.ID] = task.Clone()

		return updated
	})

	return nil
}

/*
MergeTaskClaim records an optimistic claim for a task.
*/
func (view *View) MergeTaskClaim(claim TaskClaim) error {
	if err := claim.Validate(); err != nil {
		return err
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)
		claims := updated.taskClaims[claim.TaskID]

		if claims == nil {
			claims = make(map[string]TaskClaim)
			updated.taskClaims[claim.TaskID] = claims
		}

		existing, ok := claims[claim.ActorID]
		if ok && existing.At <= claim.At {
			return updated
		}

		claims[claim.ActorID] = claim

		return updated
	})

	return nil
}

/*
MergeTaskStatus applies an A2A streaming status event into the local view.
*/
func (view *View) MergeTaskStatus(event a2a.TaskStatusUpdateEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	if _, ok := view.Task(event.TaskID); !ok {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm view task %q not found", event.TaskID),
			nil,
		)
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)
		task := updated.tasks[event.TaskID]
		task.ID = event.TaskID
		task.ContextID = event.ContextID
		task.Status = event.Status
		updated.tasks[event.TaskID] = task

		return updated
	})

	return nil
}

/*
MergeSignal records a TTL-scoped swarm signal.
*/
func (view *View) MergeSignal(signal Signal) error {
	if err := signal.Validate(); err != nil {
		return err
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)
		updated.signals = append(updated.signals, signal)

		return updated
	})

	return nil
}

/*
MergeMetric records a TTL-scoped swarm metric.
*/
func (view *View) MergeMetric(metric Metric) error {
	if err := metric.Validate(); err != nil {
		return err
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)
		updated.metrics = append(updated.metrics, metric)

		return updated
	})

	return nil
}

/*
Task returns a merged A2A task by ID.
*/
func (view *View) Task(taskID string) (a2a.Task, bool) {
	task, ok := view.state.Load().tasks[taskID]

	return task.Clone(), ok
}

/*
Tasks returns a stable snapshot of merged A2A tasks.
*/
func (view *View) Tasks() []a2a.Task {
	snapshot := view.state.Load()
	tasks := make([]a2a.Task, 0, len(snapshot.tasks))

	for _, task := range snapshot.tasks {
		tasks = append(tasks, task.Clone())
	}

	slices.SortFunc(tasks, func(firstTask, secondTask a2a.Task) int {
		return cmp.Compare(firstTask.ID, secondTask.ID)
	})

	return tasks
}

/*
TasksByState returns tasks matching an A2A lifecycle state.
*/
func (view *View) TasksByState(state a2a.TaskState) []a2a.Task {
	tasks := make([]a2a.Task, 0)

	for _, task := range view.Tasks() {
		if task.Status.State != state {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks
}

/*
SubmittedTasks returns work that no agent has marked as working yet.
*/
func (view *View) SubmittedTasks() []a2a.Task {
	return view.TasksByState(a2a.TaskStateSubmitted)
}

/*
TaskClaim returns one actor's current claim for a task.
*/
func (view *View) TaskClaim(taskID, actorID string) (TaskClaim, bool) {
	claims := view.state.Load().taskClaims[taskID]

	if claims == nil {
		return TaskClaim{}, false
	}

	claim, ok := claims[actorID]

	return claim, ok
}

/*
TaskClaims returns task claims in deterministic confirmation order.
*/
func (view *View) TaskClaims(taskID string) []TaskClaim {
	claims := view.state.Load().taskClaims[taskID]
	out := make([]TaskClaim, 0, len(claims))

	for _, claim := range claims {
		out = append(out, claim)
	}

	slices.SortFunc(out, compareTaskClaims)

	return out
}

/*
TaskClaimWinner returns the currently winning task claim.
*/
func (view *View) TaskClaimWinner(taskID string) (TaskClaim, bool) {
	claims := view.TaskClaims(taskID)

	if len(claims) == 0 {
		return TaskClaim{}, false
	}

	return claims[0], true
}

/*
RecentSignals returns a snapshot of non-expired swarm signals.
*/
func (view *View) RecentSignals() []Signal {
	return append([]Signal(nil), view.state.Load().signals...)
}

/*
SignalsByKind returns recent signals matching a kind.
*/
func (view *View) SignalsByKind(kind SignalKind) []Signal {
	signals := make([]Signal, 0)

	for _, signal := range view.RecentSignals() {
		if signal.Kind != kind {
			continue
		}

		signals = append(signals, signal)
	}

	return signals
}

/*
BlockingSignals returns recent blockers reported by swarm peers.
*/
func (view *View) BlockingSignals() []Signal {
	return view.SignalsByKind(SignalBlocker)
}

/*
RecentMetrics returns a snapshot of non-expired swarm metrics.
*/
func (view *View) RecentMetrics() []Metric {
	return append([]Metric(nil), view.state.Load().metrics...)
}

func compareTaskClaims(firstClaim, secondClaim TaskClaim) int {
	if diff := cmp.Compare(firstClaim.At, secondClaim.At); diff != 0 {
		return diff
	}

	return cmp.Compare(firstClaim.ActorID, secondClaim.ActorID)
}
