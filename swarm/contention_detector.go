package swarm

import (
	"context"
	"time"

	"github.com/theapemachine/errnie"
)

type ContentionIssueKind string

const (
	ContentionIssueStarvation ContentionIssueKind = "starvation"
	ContentionIssueDeadlock   ContentionIssueKind = "deadlock"
)

/*
ContentionDetectorOptions configures lease contention detection.
*/
type ContentionDetectorOptions struct {
	Window              time.Duration
	StarvationThreshold int
}

/*
ContentionIssue reports a detected contention failure mode.
*/
type ContentionIssue struct {
	Kind         ContentionIssueKind
	ActorID      string
	Prefix       string
	HolderID     string
	HolderPrefix string
	Attempts     int
	Since        int64
	Until        int64
	Summary      string
}

/*
ContentionDetector detects starvation and deadlock from merged contention gossip.
*/
type ContentionDetector struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	err                 error
	view                *View
	window              time.Duration
	starvationThreshold int
}

/*
NewContentionDetector instantiates contention analysis for one view.
*/
func NewContentionDetector(
	ctx context.Context,
	view *View,
	options ContentionDetectorOptions,
) (*ContentionDetector, error) {
	if options.Window <= 0 {
		return nil, errnie.Err(errnie.Validation, "swarm contention window is required", nil)
	}

	if options.StarvationThreshold <= 0 {
		return nil, errnie.Err(
			errnie.Validation,
			"swarm contention starvation threshold is required",
			nil,
		)
	}

	ctx, cancel := context.WithCancel(ctx)
	detector := &ContentionDetector{
		ctx:                 ctx,
		cancel:              cancel,
		view:                view,
		window:              options.Window,
		starvationThreshold: options.StarvationThreshold,
	}

	return detector, errnie.Require(map[string]any{
		"ctx":    detector.ctx,
		"cancel": detector.cancel,
		"view":   detector.view,
		"window": detector.window,
	})
}

/*
Detect returns current starvation and deadlock issues.
*/
func (detector *ContentionDetector) Detect(now time.Time) ([]ContentionIssue, error) {
	detector.view.PurgeExpired(now)
	contentions := detector.recentContentions(now)
	issues := detector.detectStarvation(contentions)
	issues = append(issues, detector.detectDeadlocks(contentions)...)

	return issues, nil
}

/*
Close stops detector lifecycle state.
*/
func (detector *ContentionDetector) Close() error {
	detector.cancel()

	return nil
}

func (detector *ContentionDetector) recentContentions(now time.Time) []Contention {
	cutoff := now.Add(-detector.window).UnixNano()
	contentions := make([]Contention, 0)

	for _, contention := range detector.view.RecentContentions() {
		if contention.At < cutoff {
			continue
		}

		contentions = append(contentions, contention)
	}

	return contentions
}
