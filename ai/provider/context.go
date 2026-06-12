package provider

import "context"

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
