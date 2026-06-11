package editor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theapemachine/animal/ai/tool/editor/agent"
	"github.com/theapemachine/animal/ai/tool/editor/fs"
	"github.com/theapemachine/animal/ai/tool/editor/workspace"
	"github.com/theapemachine/animal/config"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Server exposes the virtual workspace editor over MCP.
*/
type Server struct {
	ctx         context.Context
	cancel      context.CancelFunc
	pool        *qpool.Q[any]
	app         *fiber.App
	root        string
	coordinator *lease.Coordinator
	document    Document
}

/*
NewServer instantiates an editor MCP server bound to the resolved workspace.
*/
func NewServer(ctx context.Context, pool *qpool.Q[any]) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	root, err := workspace.Resolve()
	if err != nil {
		cancel()
		return nil, err
	}

	coordinatorOptions, err := config.LeaseCoordinatorOptionsFromViper(lease.PathKeySpace{})
	if err != nil {
		cancel()
		return nil, err
	}

	coordinator, err := lease.NewCoordinator(coordinatorOptions)
	if err != nil {
		cancel()
		return nil, err
	}

	document, err := fs.NewDocument(root, coordinator)
	if err != nil {
		cancel()
		return nil, err
	}

	server := &Server{
		ctx:         ctx,
		cancel:      cancel,
		pool:        pool,
		app:         fiber.New(),
		root:        root,
		coordinator: coordinator,
		document:    document,
	}

	return server, errnie.Require(map[string]any{
		"ctx":         ctx,
		"cancel":      cancel,
		"pool":        pool,
		"app":         server.app,
		"root":        root,
		"coordinator": coordinator,
		"document":    document,
	})
}

/*
Run serves MCP tools for read, search, and replace over SSE.
*/
func (server *Server) Run() error {
	handler := mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		access := agent.ParseHeaders(request.Header)
		return server.mcpServer(access)
	}, nil)

	server.app.All("/mcp/editor", adaptor.HTTPHandler(handler))
	return server.app.Listen(":3000")
}

func (server *Server) mcpServer(access agent.Access) *mcp.Server {
	sessionServer := mcp.NewServer(&mcp.Implementation{Name: "editor"}, nil)

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "read_file",
		Description: "Read a workspace file with optional 1-based line range. Content is returned with line numbers.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args ReadParams,
	) (*mcp.CallToolResult, ReadResult, error) {
		return server.readFile(ctx, access, args)
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "search",
		Description: "Search a workspace file for a regular expression and return numbered matching lines.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args SearchParams,
	) (*mcp.CallToolResult, SearchResult, error) {
		return server.search(ctx, access, args)
	})

	mcp.AddTool(sessionServer, &mcp.Tool{
		Name:        "replace",
		Description: "Replace a unique exact string in a workspace file. Fails when the match is missing or ambiguous.",
	}, func(
		ctx context.Context, _ *mcp.CallToolRequest, args ReplaceParams,
	) (*mcp.CallToolResult, ReplaceResult, error) {
		return server.replace(ctx, access, args)
	})

	return sessionServer
}

func (server *Server) readFile(
	ctx context.Context,
	access agent.Access,
	args ReadParams,
) (*mcp.CallToolResult, ReadResult, error) {
	if notice, ok := server.readNotice(args.Path, access); ok {
		return nil, notice, nil
	}

	result, err := server.document.Read(ctx, args)
	if err != nil {
		return nil, ReadResult{}, err
	}

	return nil, result, nil
}

func (server *Server) search(
	ctx context.Context,
	access agent.Access,
	args SearchParams,
) (*mcp.CallToolResult, SearchResult, error) {
	if notice, ok := server.readNotice(args.Path, access); ok {
		return nil, SearchResult{Changing: notice.Changing}, nil
	}

	result, err := server.document.Search(ctx, args)
	if err != nil {
		return nil, SearchResult{}, err
	}

	return nil, result, nil
}

func (server *Server) readNotice(path string, access agent.Access) (ReadResult, bool) {
	readErr := server.coordinator.ObserveRead(path, principalFromAccess(access))

	changing, ok := lease.AsChanging(readErr)
	if !ok {
		return ReadResult{}, false
	}

	return ReadResult{
		Changing: fileChangingNotice(changing),
	}, true
}

func fileChangingNotice(changing *lease.ChangingError) *FileChangingNotice {
	return &FileChangingNotice{
		Path:        changing.Key,
		LeasePrefix: changing.LeaseKey,
		AgentID:     changing.ActorID,
		Message:     changing.Error(),
	}
}

func (server *Server) replace(
	ctx context.Context,
	access agent.Access,
	args ReplaceParams,
) (*mcp.CallToolResult, ReplaceResult, error) {
	if err := server.coordinator.CanWrite(args.Path, principalFromAccess(access)); err != nil {
		return nil, ReplaceResult{}, err
	}

	if err := server.document.Replace(ctx, args); err != nil {
		return nil, ReplaceResult{}, err
	}

	return nil, ReplaceResult{Path: args.Path}, nil
}

/*
WorkspaceRoot exposes the resolved sandbox path for diagnostics.
*/
func (server *Server) WorkspaceRoot() string {
	return server.root
}

/*
LeaseCoordinator exposes the in-process lease manager.
*/
func (server *Server) LeaseCoordinator() *lease.Coordinator {
	return server.coordinator
}

/*
AcquireLease grants an agent exclusive access to a path prefix.
*/
func (server *Server) AcquireLease(prefix, agentID string) error {
	return server.coordinator.AcquireID(prefix, agentID)
}

/*
ReleaseLease drops a path-prefix lease held by an agent.
*/
func (server *Server) ReleaseLease(prefix, agentID string) error {
	return server.coordinator.ReleaseID(prefix, agentID)
}

/*
Close stops the server context.
*/
func (server *Server) Close() {
	server.cancel()
}

/*
String returns a short diagnostic label.
*/
func (server *Server) String() string {
	return fmt.Sprintf("editor@%s", server.root)
}
