package swarm

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/theapemachine/errnie"
)

const (
	MessageTypeRumor      = "rumor"
	MessageTypeTask       = "task"
	MessageTypeTaskStatus = "task_status"
	MessageTypeSignal     = "signal"
	MessageTypeMetric     = "metric"
)

/*
SignalKind classifies a swarm broadcast that should attract available attention.
*/
type SignalKind string

const (
	SignalFriction    SignalKind = "friction"
	SignalQuality     SignalKind = "quality"
	SignalBlocker     SignalKind = "blocker"
	SignalOpportunity SignalKind = "opportunity"
)

/*
Signal is a non-hierarchical broadcast about friction, quality, blockers, or opportunity.
*/
type Signal struct {
	ID        string
	GoalID    string
	TaskID    string
	ActorID   string
	ActorName string
	Role      string
	Kind      SignalKind
	Summary   string
	Detail    string
	At        int64
}

/*
Metric records a normalized success signal that can feed training data extraction.
*/
type Metric struct {
	ID        string
	GoalID    string
	TaskID    string
	ActorID   string
	ActorName string
	Role      string
	Name      string
	Score     float64
	Success   bool
	Evidence  string
	At        int64
}

/*
NewSignalAt builds a signal stamped with now.
*/
func NewSignalAt(
	kind SignalKind,
	actorID, actorName, role string,
	at time.Time,
) Signal {
	return Signal{
		ActorID:   actorID,
		ActorName: actorName,
		Role:      role,
		Kind:      kind,
		At:        at.UnixNano(),
	}
}

/*
NewMetricAt builds a metric stamped with now.
*/
func NewMetricAt(
	actorID, actorName, role string,
	at time.Time,
) Metric {
	return Metric{
		ActorID:   actorID,
		ActorName: actorName,
		Role:      role,
		At:        at.UnixNano(),
	}
}

/*
Validate checks required signal fields.
*/
func (signal Signal) Validate() error {
	if strings.TrimSpace(signal.ActorID) == "" {
		return errnie.Err(errnie.Validation, "swarm signal actor ID is required", nil)
	}

	if !validSignalKind(signal.Kind) {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm signal kind %q is invalid", signal.Kind),
			nil,
		)
	}

	if strings.TrimSpace(signal.Summary) == "" {
		return errnie.Err(errnie.Validation, "swarm signal summary is required", nil)
	}

	if signal.At <= 0 {
		return errnie.Err(errnie.Validation, "swarm signal timestamp is required", nil)
	}

	return nil
}

/*
Validate checks required metric fields.
*/
func (metric Metric) Validate() error {
	if strings.TrimSpace(metric.ActorID) == "" {
		return errnie.Err(errnie.Validation, "swarm metric actor ID is required", nil)
	}

	if strings.TrimSpace(metric.Name) == "" {
		return errnie.Err(errnie.Validation, "swarm metric name is required", nil)
	}

	if math.IsNaN(metric.Score) || math.IsInf(metric.Score, 0) {
		return errnie.Err(errnie.Validation, "swarm metric score must be finite", nil)
	}

	if metric.Score < 0 || metric.Score > 1 {
		return errnie.Err(errnie.Validation, "swarm metric score must be between 0 and 1", nil)
	}

	if metric.At <= 0 {
		return errnie.Err(errnie.Validation, "swarm metric timestamp is required", nil)
	}

	return nil
}

func validSignalKind(kind SignalKind) bool {
	switch kind {
	case SignalFriction, SignalQuality, SignalBlocker, SignalOpportunity:
		return true
	default:
		return false
	}
}
