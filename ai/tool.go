package ai

import "io"

type Tool interface {
	io.ReadWriteCloser
}
