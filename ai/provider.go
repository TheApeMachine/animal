package ai

import "io"

/*
Provider is a model backend that streams completions into the shared qpool bus.
ReadWriteCloser ties provider lifetime to the agent cycle so endpoints shut down with their parent context.
*/
type Provider interface {
	io.ReadWriteCloser
}
