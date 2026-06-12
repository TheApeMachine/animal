package ai

import "io"

/*
Tool is an MCP-backed capability registered on an agent before a workflow step runs.
ReadWriteCloser exposes a uniform attach/detach contract for servers such as the workspace editor or browser.
*/
type Tool interface {
	io.ReadWriteCloser
}
