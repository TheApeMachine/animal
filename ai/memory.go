package ai

import "io"

/*
Memory is a pluggable recall surface attached to an agent for the duration of a run.
ReadWriteCloser lets orchestration open, stream from, and tear down memory backends uniformly.
*/
type Memory interface {
	io.ReadWriteCloser
}
