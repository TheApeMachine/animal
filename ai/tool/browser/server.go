package browser

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theapemachine/animal/ai/tool/browser/doc"
	browserrod "github.com/theapemachine/animal/ai/tool/browser/rod"
	"github.com/theapemachine/qpool"
)

/*
Server exposes a stealth headless browser over MCP.
*/
type Server struct {
	ctx        context.Context
	cancel     context.CancelFunc
	app        *fiber.App
	config     Config
	session    *browserrod.Session
	worker     *qpool.Q[any]
	closed     atomic.Bool
	commandSeq atomic.Uint64
}

/*
NewServer launches the browser and prepares MCP handlers.
*/
func NewServer(ctx context.Context, config Config) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	session, err := Open(config)
	if err != nil {
		cancel()
		return nil, err
	}

	server := &Server{
		ctx:     ctx,
		cancel:  cancel,
		app:     fiber.New(),
		config:  config,
		session: session,
		worker:  qpool.NewQ[any](ctx, 1, 1, nil),
	}

	return server, nil
}

/*
MCPServer returns an MCP server wired to the shared browser session.
*/
func (server *Server) MCPServer() *mcp.Server {
	sessionServer := mcp.NewServer(&mcp.Implementation{Name: "browser", Version: "v1.0.0"}, nil)

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "browser_navigate",
		Description: "Navigate the headless browser to an absolute URL and wait for the page to load.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args doc.NavigateParams,
	) (*mcp.CallToolResult, doc.NavigateResult, error) {
		return withSession(server, ctx, func(session doc.Browser) (doc.NavigateResult, error) {
			return session.Navigate(ctx, args)
		})
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "browser_evaluate",
		Description: "Evaluate JavaScript in the current page and return the JSON-serialized result.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args doc.EvaluateParams,
	) (*mcp.CallToolResult, doc.EvaluateResult, error) {
		return withSession(server, ctx, func(session doc.Browser) (doc.EvaluateResult, error) {
			return session.Evaluate(ctx, args)
		})
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "browser_content",
		Description: "Read the current page as visible text or HTML. Large pages are truncated.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args doc.ContentParams,
	) (*mcp.CallToolResult, doc.ContentResult, error) {
		return withSession(server, ctx, func(session doc.Browser) (doc.ContentResult, error) {
			return session.Content(ctx, args)
		})
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "browser_click",
		Description: "Click an element matched by CSS selector and wait for navigation to settle.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args doc.ClickParams,
	) (*mcp.CallToolResult, doc.ClickResult, error) {
		return withSession(server, ctx, func(session doc.Browser) (doc.ClickResult, error) {
			return session.Click(ctx, args)
		})
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "browser_wait",
		Description: "Wait for a CSS selector to appear or for the page load event to complete.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args doc.WaitParams,
	) (*mcp.CallToolResult, doc.WaitResult, error) {
		return withSession(server, ctx, func(session doc.Browser) (doc.WaitResult, error) {
			return session.Wait(ctx, args)
		})
	})

	return sessionServer
}

func withSession[T any](
	server *Server,
	ctx context.Context,
	action func(doc.Browser) (T, error),
) (*mcp.CallToolResult, T, error) {
	var zero T

	if server.closed.Load() {
		return nil, zero, fmt.Errorf("browser: session is closed")
	}

	jobID := fmt.Sprintf("browser-%d", server.commandSeq.Add(1))
	wait := server.worker.Schedule(jobID, func(_ context.Context) (any, error) {
		if server.session == nil {
			return nil, fmt.Errorf("browser: session is closed")
		}

		return action(server.session)
	})

	result, err := wait.Get(ctx)
	if err != nil {
		return nil, zero, err
	}

	if result.Error != nil {
		return nil, zero, result.Error
	}

	typed, ok := result.Value.(T)
	if !ok {
		return nil, zero, fmt.Errorf("browser: session action returned unexpected type %T", result.Value)
	}

	return nil, typed, nil
}

/*
Run serves MCP browser tools over SSE on :3001/mcp/browser.
*/
func (server *Server) Run() error {
	handler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return server.MCPServer()
	}, nil)

	server.app.All("/mcp/browser", adaptor.HTTPHandler(handler))
	return server.app.Listen(":3001")
}

/*
Close stops the browser and server context.
*/
func (server *Server) Close() error {
	if server.closed.Swap(true) {
		return nil
	}

	jobID := fmt.Sprintf("browser-close-%d", server.commandSeq.Add(1))
	wait := server.worker.Schedule(jobID, func(_ context.Context) (any, error) {
		if server.session == nil {
			return nil, nil
		}

		closeErr := server.session.Close()
		server.session = nil

		return nil, closeErr
	})

	_, err := wait.Get(server.ctx)
	server.cancel()
	server.worker.Close()

	return err
}
