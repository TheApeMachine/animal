package ai

import "io"

type Memory interface {
	io.ReadWriteCloser
}
