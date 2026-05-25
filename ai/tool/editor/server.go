package editor

import (
	"context"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Server sets up the MCP server for the tool/
*/
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	pool   *qpool.Q
	app    *fiber.App
}

func NewServer(ctx context.Context, pool *qpool.Q) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	server := &Server{
		ctx:    ctx,
		cancel: cancel,
		pool:   pool,
		app:    fiber.New(),
	}

	return server, errnie.Require(map[string]any{
		"ctx":    ctx,
		"cancel": cancel,
		"pool":   pool,
		"app":    server.app,
	})
}

func (server *Server) Run() error {
	srv := mcp.NewServer(&mcp.Implementation{Name: "editor"}, nil)

	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name:        "read_file",
			Description: "read a file",
		}, server.ReadFile,
	)

	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		url := request.URL.Path

		switch url {
		case "/read_file":
			return srv
		default:
			return nil
		}
	}, nil)

	server.app.Get("/mcp/editor", handler)
	return server.app.Listen(":3000")
}

func (server *Server) ReadFile(
	ctx context.Context, req *mcp.CallToolRequest, args ReadParams,
) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		StructuredContent: ReadResult{
			Content: "Hi",
		},
	}, nil, nil
}
