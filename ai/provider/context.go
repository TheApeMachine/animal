package provider

import "context"

type Context struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	Messages []Message
}

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
