package enricher

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/theapemachine/animal/ai/agent"
	"github.com/theapemachine/animal/ai/mcpclient"
	"github.com/theapemachine/animal/ai/tool/browser"
)

/*
LLMConfig selects the OpenAI-compatible model endpoint.
*/
type LLMConfig struct {
	Endpoint string
	APIKey   string
	Model    string
	MaxSteps int
}

/*
Session runs an MCP-backed browser agent for autonomous web enrichment.
*/
type Session struct {
	browserServer *browser.Server
	mcpSession    *mcp.ClientSession
	runner        *agent.Runner
}

/*
Open wires a stealth browser MCP server to an in-process agent runner.
*/
func Open(
	ctx context.Context,
	llmConfig LLMConfig,
	browserConfig browser.Config,
) (*Session, error) {
	browserServer, err := browser.NewServer(ctx, browserConfig)
	if err != nil {
		return nil, fmt.Errorf("browser server: %w", err)
	}

	mcpSession, err := mcpclient.ConnectInMemory(ctx, browserServer.MCPServer())
	if err != nil {
		_ = browserServer.Close()
		return nil, fmt.Errorf("browser mcp: %w", err)
	}

	return &Session{
		browserServer: browserServer,
		mcpSession:    mcpSession,
		runner: agent.NewRunner(
			llmConfig.Endpoint,
			llmConfig.APIKey,
			llmConfig.Model,
			mcpSession,
			llmConfig.MaxSteps,
		),
	}, nil
}

/*
Run executes the agent loop until the model returns final text.
*/
func (session *Session) Run(
	ctx context.Context,
	system string,
	user string,
) (string, error) {
	return session.runner.Run(ctx, system, user)
}

/*
Close shuts down the MCP client and browser.
*/
func (session *Session) Close() error {
	if session.mcpSession != nil {
		_ = session.mcpSession.Close()
	}

	if session.browserServer != nil {
		return session.browserServer.Close()
	}

	return nil
}
