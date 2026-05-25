package ai

import "io"

type Provider interface {
	io.ReadWriteCloser
}
