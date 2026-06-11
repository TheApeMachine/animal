// Start the virtual workspace MCP editor over SSE on :3000.
//
// Run from the repository root:
//
//	make example-editor-mcp
//
// Point ANIMAL_AGENT_WORKSPACE at a sandbox directory before starting.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/theapemachine/animal/ai/tool/editor"
	"github.com/theapemachine/animal/examples/support"
)

func main() {
	if loadErr := support.LoadViper(); loadErr != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", loadErr)
		os.Exit(1)
	}

	ctx := context.Background()
	pool := support.NewQPool(ctx)

	server, err := editor.NewServer(ctx, pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()

	fmt.Printf("editor workspace: %s\n", server.WorkspaceRoot())
	fmt.Println("MCP SSE endpoint: http://127.0.0.1:3000/mcp/editor")
	fmt.Println("agent headers: X-Agent-ID, X-Agent-Require-Lease, X-Agent-Lease-Prefixes")

	if runErr := server.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", runErr)
		os.Exit(1)
	}
}
