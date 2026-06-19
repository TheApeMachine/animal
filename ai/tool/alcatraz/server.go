package alcatraz

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theapemachine/errnie"
)

/*
Server exposes Bridge read/write operations as MCP tools.
*/
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	bridge *Bridge
}

/*
ReadParams configures one environment stdout/stderr read.
*/
type ReadParams struct {
	MaxBytes int `json:"max_bytes,omitempty"`
}

/*
ReadResult returns environment stdout/stderr as prompt input.
*/
type ReadResult struct {
	Content string `json:"content"`
	Bytes   int    `json:"bytes"`
}

/*
WriteParams carries assistant output to environment stdin.
*/
type WriteParams struct {
	Content string `json:"content"`
}

/*
WriteResult reports bytes written to environment stdin.
*/
type WriteResult struct {
	Bytes int `json:"bytes"`
}

/*
NewServer instantiates an MCP server adapter over a Bridge.
*/
func NewServer(ctx context.Context, bridge *Bridge) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	server := &Server{
		ctx:    ctx,
		cancel: cancel,
		bridge: bridge,
	}

	return server, errnie.Require(map[string]any{
		"ctx":    server.ctx,
		"cancel": server.cancel,
		"bridge": server.bridge,
	})
}

/*
MCPServer returns a tool server for interactive Linux environment stdio.
*/
func (server *Server) MCPServer() *mcp.Server {
	sessionServer := mcp.NewServer(
		&mcp.Implementation{Name: "alcatraz", Version: "v1.0.0"},
		nil,
	)

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "alcatraz_read",
		Description: "Read stdout/stderr from the attached Linux environment.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args ReadParams,
	) (*mcp.CallToolResult, ReadResult, error) {
		return server.readTool(ctx, args)
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "alcatraz_write",
		Description: "Write assistant output to stdin of the attached Linux environment.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args WriteParams,
	) (*mcp.CallToolResult, WriteResult, error) {
		return server.writeTool(ctx, args)
	})

	return sessionServer
}

func (server *Server) readTool(
	ctx context.Context,
	args ReadParams,
) (*mcp.CallToolResult, ReadResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, ReadResult{}, err
	}

	maxBytes := args.MaxBytes

	if maxBytes == 0 {
		maxBytes = server.bridge.bufferSize
	}

	message, err := server.bridge.ReadPromptN(maxBytes)
	if err != nil {
		return nil, ReadResult{}, err
	}

	return nil, ReadResult{
		Content: message.Content,
		Bytes:   len(message.Content),
	}, nil
}

func (server *Server) writeTool(
	ctx context.Context,
	args WriteParams,
) (*mcp.CallToolResult, WriteResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, WriteResult{}, err
	}

	count, err := server.bridge.writeContent(args.Content)
	if err != nil {
		return nil, WriteResult{}, err
	}

	return nil, WriteResult{
		Bytes: count,
	}, nil
}

/*
Close cancels the MCP server adapter.
*/
func (server *Server) Close() error {
	server.cancel()

	return nil
}
