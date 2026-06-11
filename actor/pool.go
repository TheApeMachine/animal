package actor

import (
	"context"

	"github.com/theapemachine/errnie"
)

/*
Pool manages a collection of actors and their relationships.
*/
type Pool struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	actors map[string]*Identity
}

/*
NewPool creates a new pool with a context.
*/
func NewPool(ctx context.Context) (*Pool, error) {
	ctx, cancel := context.WithCancel(ctx)

	pool := &Pool{
		ctx:    ctx,
		cancel: cancel,
		err:    nil,
		actors: make(map[string]*Identity),
	}

	return pool, errnie.Require(map[string]any{
		"ctx":    ctx,
		"cancel": cancel,
		"err":    nil,
		"actors": make(map[string]*Identity),
	})
}
