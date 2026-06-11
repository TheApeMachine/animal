package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/theapemachine/animal/ai/mcpclient"
	"github.com/theapemachine/animal/ai/provider"
)

/*
Runner executes an LLM agent loop against MCP tools.
*/
type Runner struct {
	client   openaiapi.Client
	model    string
	session  *mcp.ClientSession
	maxSteps int
}

/*
NewRunner wires an OpenAI-compatible client to an MCP client session.
*/
func NewRunner(
	endpoint, apiKey, model string,
	session *mcp.ClientSession,
	maxSteps int,
) *Runner {
	if maxSteps <= 0 {
		maxSteps = 12
	}

	client := openaiapi.NewClient(
		option.WithBaseURL(strings.TrimRight(endpoint, "/")),
		option.WithAPIKey(apiKey),
	)

	return &Runner{
		client:   client,
		model:    model,
		session:  session,
		maxSteps: maxSteps,
	}
}

/*
Run executes tool-augmented completion until the model returns final text.
*/
func (runner *Runner) Run(
	ctx context.Context,
	system string,
	user string,
) (string, error) {
	tools, err := runner.listTools(ctx)
	if err != nil {
		return "", err
	}

	agentContext := provider.Context{
		Messages: []provider.Message{
			{Role: "user", Content: user},
		},
	}

	for step := 0; step < runner.maxSteps; step++ {
		response, err := completeWithTools(
			ctx,
			runner.client,
			runner.model,
			system,
			&agentContext,
			tools,
		)
		if err != nil {
			return "", err
		}

		if len(response.ToolCalls) == 0 {
			if strings.TrimSpace(response.Content) == "" {
				return "", fmt.Errorf("agent: empty completion")
			}

			return response.Content, nil
		}

		agentContext.Messages = append(agentContext.Messages, provider.Message{
			Role:      "assistant",
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})

		for _, toolCall := range response.ToolCalls {
			toolOutput, callErr := runner.callTool(ctx, toolCall)
			if callErr != nil {
				toolOutput = fmt.Sprintf(`{"error":%q}`, callErr.Error())
			}

			agentContext.Messages = append(agentContext.Messages, provider.Message{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Name:       toolCall.Name,
				Content:    toolOutput,
			})
		}
	}

	return "", fmt.Errorf("agent: exceeded max steps (%d)", runner.maxSteps)
}

func (runner *Runner) listTools(ctx context.Context) ([]provider.ToolDefinition, error) {
	list, err := runner.session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	tools := make([]provider.ToolDefinition, 0, len(list.Tools))

	for _, tool := range list.Tools {
		parameters, marshalErr := json.Marshal(tool.InputSchema)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal tool schema %s: %w", tool.Name, marshalErr)
		}

		tools = append(tools, provider.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  parameters,
		})
	}

	return tools, nil
}

func (runner *Runner) callTool(
	ctx context.Context,
	toolCall provider.ToolCall,
) (string, error) {
	var arguments map[string]any

	if strings.TrimSpace(toolCall.Arguments) != "" {
		if err := json.Unmarshal([]byte(toolCall.Arguments), &arguments); err != nil {
			return "", fmt.Errorf("decode tool arguments: %w", err)
		}
	}

	payload, err := mcpclient.CallToolJSON(ctx, runner.session, toolCall.Name, arguments)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}
