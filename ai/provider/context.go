package provider

import (
	"context"

	"github.com/theapemachine/errnie"
)

/*
Context accumulates the rolling message history for one agent or completion request.
It owns a cancellable scope so in-flight provider work stops when the parent orchestration exits.
*/
type Context struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	Messages []Message
}

/*
contextOptions configures optional fields when constructing a provider Context.
Functional options keep NewContext stable as new conversation-scoped settings are added.
*/
type contextOptions func(*Context)

func NewContext(ctx context.Context, opts ...contextOptions) *Context {
	ctx, cancel := context.WithCancel(ctx)

	return &Context{
		ctx:      ctx,
		cancel:   cancel,
		err:      nil,
		Messages: make([]Message, 0),
	}
}

/*
Clone copies the message history into a new cancellable context scope.
*/
func (agentCtx *Context) Clone(ctx context.Context) (*Context, error) {
	if agentCtx == nil {
		return nil, errnie.Err(
			errnie.Validation,
			"provider context is required",
			nil,
		)
	}

	clone := NewContext(ctx)
	clone.Messages = append(clone.Messages, agentCtx.Messages...)

	return clone, nil
}

/*
Append adds one message to the context history.
*/
func (agentCtx *Context) Append(message Message) error {
	if agentCtx == nil {
		return errnie.Err(
			errnie.Validation,
			"provider context is required",
			nil,
		)
	}

	agentCtx.Messages = append(agentCtx.Messages, message)

	return nil
}

/*
Replace swaps the entire message history.
*/
func (agentCtx *Context) Replace(messages []Message) error {
	if agentCtx == nil {
		return errnie.Err(
			errnie.Validation,
			"provider context is required",
			nil,
		)
	}

	agentCtx.Messages = append(agentCtx.Messages[:0], messages...)

	return nil
}
