package actor

import (
	"context"

	"github.com/theapemachine/errnie"
)

/*
Identity is the stable actor handle used for lease principals
and swarm gossip. It carries a cancellable context so actor
lifetimes propagate through the orchestration tree.
*/
type Identity struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	ID     string
}

/*
NewIdentity creates a new identity with a cancellable context.
*/
func NewIdentity(ctx context.Context, id string) (*Identity, error) {
	ctx, cancel := context.WithCancel(ctx)

	identity := &Identity{
		ctx:    ctx,
		cancel: cancel,
		err:    nil,
		ID:     id,
	}

	return identity, errnie.Require(map[string]any{
		"ctx":    ctx,
		"cancel": cancel,
		"err":    nil,
		"ID":     id,
	})
}
