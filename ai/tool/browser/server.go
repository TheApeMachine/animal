package browser

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theapemachine/animal/ai/tool/browser/doc"
	browserrod "github.com/theapemachine/animal/ai/tool/browser/rod"
)

/*
Server exposes a stealth headless browser over MCP.
*/
type Server struct {
	ctx       context.Context
	cancel    context.CancelFunc
	app       *fiber.App
	config    Config
	session   *browserrod.Session
	sessionMu sync.Mutex
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

	server.sessionMu.Lock()
	defer server.sessionMu.Unlock()

	if server.session == nil {
		return nil, zero, fmt.Errorf("browser: session is closed")
	}

	result, err := action(server.session)
	if err != nil {
		return nil, zero, err
	}

	return nil, result, nil
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
	server.sessionMu.Lock()
	defer server.sessionMu.Unlock()

	if server.session != nil {
		_ = server.session.Close()
		server.session = nil
	}

	server.cancel()
	return nil
}
