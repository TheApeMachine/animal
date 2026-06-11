package actor

import (
	"context"

	"github.com/theapemachine/errnie"
)

type Identity struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	ID     string
}

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
