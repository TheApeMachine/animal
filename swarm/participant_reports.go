package swarm

import "time"

/*
ReportSignal broadcasts friction, quality, blocker, or opportunity information.
*/
func (participant *Participant) ReportSignal(
	kind SignalKind,
	goalID, taskID, summary, detail string,
) error {
	signal := NewSignalAt(
		kind,
		participant.actorID,
		participant.actorName,
		participant.role,
		time.Now(),
	)
	signal.GoalID = goalID
	signal.TaskID = taskID
	signal.Summary = summary
	signal.Detail = detail

	if err := participant.view.MergeSignal(signal); err != nil {
		return err
	}

	return participant.mesh.PublishValue(participant.actorID, MessageTypeSignal, signal)
}

/*
ReportMetric broadcasts a normalized success metric for a task or goal.
*/
func (participant *Participant) ReportMetric(
	goalID, taskID, name string,
	score float64,
	success bool,
	evidence string,
) error {
	metric := NewMetricAt(
		participant.actorID,
		participant.actorName,
		participant.role,
		time.Now(),
	)
	metric.GoalID = goalID
	metric.TaskID = taskID
	metric.Name = name
	metric.Score = score
	metric.Success = success
	metric.Evidence = evidence

	if err := participant.view.MergeMetric(metric); err != nil {
		return err
	}

	return participant.mesh.PublishValue(participant.actorID, MessageTypeMetric, metric)
}
