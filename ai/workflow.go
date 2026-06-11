package ai

import (
	"context"

	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Workflow coordinates multi-agent collaboration using qpool jobs and optional broadcast leases.
*/
type Workflow struct {
	ctx    context.Context
	cancel context.CancelFunc
	pool   *qpool.Q[any]
}

func NewWorkflow(ctx context.Context, pool *qpool.Q[any]) (*Workflow, error) {
	ctx, cancel := context.WithCancel(ctx)

	wf := &Workflow{
		ctx:    ctx,
		cancel: cancel,
		pool:   pool,
	}

	return wf, errnie.Require(map[string]any{
		"ctx":  ctx,
		"pool": pool,
	})
}
