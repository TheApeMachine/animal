package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

/*
ConnectInMemory pairs a client session with an MCP server over memory transport.
*/
func ConnectInMemory(
	ctx context.Context,
	server *mcp.Server,
) (*mcp.ClientSession, error) {
	transportA, transportB := mcp.NewInMemoryTransports()

	if _, err := server.Connect(ctx, transportA, nil); err != nil {
		return nil, fmt.Errorf("mcp server connect: %w", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "animal-agent", Version: "v1.0.0"}, nil)

	session, err := client.Connect(ctx, transportB, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp client connect: %w", err)
	}

	return session, nil
}

/*
CallToolJSON invokes an MCP tool and returns the structured output as JSON bytes.
*/
func CallToolJSON(
	ctx context.Context,
	session *mcp.ClientSession,
	name string,
	arguments any,
) ([]byte, error) {
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return nil, err
	}

	if result.IsError {
		return nil, fmt.Errorf("tool %s failed: %s", name, toolResultText(result))
	}

	if result.StructuredContent != nil {
		switch content := result.StructuredContent.(type) {
		case json.RawMessage:
			return content, nil
		default:
			return json.Marshal(content)
		}
	}

	return json.Marshal(map[string]string{"result": toolResultText(result)})
}

func toolResultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}

	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			return text.Text
		}
	}

	return ""
}
