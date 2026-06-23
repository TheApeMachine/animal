package swarm

import (
	"context"
	"time"

	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/errnie"
)

/*
DashboardOptions configures the live swarm dashboard handler.
*/
type DashboardOptions struct {
	Refresh time.Duration
}

/*
DashboardSnapshot is the JSON read model behind the live swarm dashboard.
*/
type DashboardSnapshot struct {
	GeneratedAt int64
	Claims      []ClaimRecord
	Announces   []AnnounceRecord
	Statuses    []StatusRecord
	Tasks       []a2a.Task
	TaskClaims  []TaskClaim
	Signals     []Signal
	Metrics     []Metric
	Contentions []Contention
}

/*
Dashboard exposes a live HTTP view of swarm gossip and task state.
*/
type Dashboard struct {
	ctx     context.Context
	cancel  context.CancelFunc
	err     error
	view    *View
	refresh time.Duration
}

/*
NewDashboard instantiates a dashboard over one swarm view.
*/
func NewDashboard(
	ctx context.Context,
	view *View,
	options DashboardOptions,
) (*Dashboard, error) {
	ctx, cancel := context.WithCancel(ctx)

	dashboard := &Dashboard{
		ctx:     ctx,
		cancel:  cancel,
		view:    view,
		refresh: options.Refresh,
	}

	err := errnie.Require(map[string]any{
		"ctx":    dashboard.ctx,
		"cancel": dashboard.cancel,
		"view":   dashboard.view,
	})

	if err != nil {
		cancel()

		return nil, err
	}

	if dashboard.refresh <= 0 {
		cancel()

		return nil, errnie.Err(
			errnie.Validation,
			"swarm dashboard refresh is required",
			nil,
		)
	}

	return dashboard, nil
}

/*
Snapshot returns a stable dashboard read model.
*/
func (dashboard *Dashboard) Snapshot() DashboardSnapshot {
	dashboard.view.PurgeExpired(time.Now())

	return DashboardSnapshot{
		GeneratedAt: time.Now().UnixNano(),
		Claims:      dashboard.view.Claims(),
		Announces:   dashboard.view.RecentAnnounces(),
		Statuses:    dashboard.view.Statuses(),
		Tasks:       dashboard.view.Tasks(),
		TaskClaims:  dashboard.view.AllTaskClaims(),
		Signals:     dashboard.view.RecentSignals(),
		Metrics:     dashboard.view.RecentMetrics(),
		Contentions: dashboard.view.RecentContentions(),
	}
}

/*
Close stops dashboard streaming loops.
*/
func (dashboard *Dashboard) Close() error {
	dashboard.cancel()

	return nil
}
