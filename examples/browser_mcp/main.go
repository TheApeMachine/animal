// Start the stealth browser MCP server over SSE on :3001.
//
// Run from the repository root:
//
//	make example-browser-mcp
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/theapemachine/animal/ai/tool/browser"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := browser.NewServer(ctx, browser.DefaultConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
		os.Exit(1)
	}
	defer server.Close()

	fmt.Println("MCP SSE endpoint: http://127.0.0.1:3001/mcp/browser")
	fmt.Println("Tools: browser_navigate, browser_evaluate, browser_content, browser_click, browser_wait")

	if runErr := server.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", runErr)
		os.Exit(1)
	}
}
