package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/theapemachine/animal/ai/provider"
)

func responseTools(tools []provider.ToolDefinition) []responses.ToolUnionParam {
	responseTools := make([]responses.ToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		var parameters map[string]any

		if len(tool.Parameters) > 0 {
			_ = json.Unmarshal(tool.Parameters, &parameters)
		}

		if parameters == nil {
			parameters = map[string]any{}
		}

		responseTools = append(responseTools, responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        tool.Name,
				Description: openaiapi.String(tool.Description),
				Parameters:  parameters,
				Strict:      openaiapi.Bool(true),
			},
		})
	}

	return responseTools
}

func completeWithTools(
	ctx context.Context,
	client openaiapi.Client,
	model string,
	system string,
	agentCtx *provider.Context,
	tools []provider.ToolDefinition,
) (provider.ToolCompletion, error) {
	if agentCtx == nil {
		return provider.ToolCompletion{}, fmt.Errorf("agent: context is required")
	}

	inputItems, err := responseInputItems(agentCtx.Messages)
	if err != nil {
		return provider.ToolCompletion{}, err
	}

	request := responses.ResponseNewParams{
		Model: model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
		Tools: responseTools(tools),
	}

	if strings.TrimSpace(system) != "" {
		request.Instructions = param.NewOpt(system)
	}

	response, err := client.Responses.New(ctx, request)
	if err != nil {
		return provider.ToolCompletion{}, fmt.Errorf("agent: responses create: %w", err)
	}

	completion := toolCompletionFromResponse(response)
	if strings.TrimSpace(completion.Content) == "" && len(completion.ToolCalls) == 0 {
		return provider.ToolCompletion{}, fmt.Errorf("agent: responses create returned no output")
	}

	return completion, nil
}

func toolCompletionFromResponse(response *responses.Response) provider.ToolCompletion {
	completion := provider.ToolCompletion{}

	for _, item := range response.Output {
		switch variant := item.AsAny().(type) {
		case responses.ResponseOutputMessage:
			for _, content := range variant.Content {
				if content.Type == "output_text" {
					completion.Content += content.Text
				}
			}
		case responses.ResponseFunctionToolCall:
			completion.ToolCalls = append(completion.ToolCalls, provider.ToolCall{
				ID:        variant.CallID,
				Name:      variant.Name,
				Arguments: variant.Arguments,
			})
		}
	}

	return completion
}

func responseInputItems(messages []provider.Message) (responses.ResponseInputParam, error) {
	input := make(responses.ResponseInputParam, 0, len(messages))

	for _, message := range messages {
		items, mapErr := responseInputItemsForMessage(message)
		if mapErr != nil {
			return nil, mapErr
		}

		input = append(input, items...)
	}

	if len(input) == 0 {
		return nil, fmt.Errorf("agent: at least one message is required")
	}

	return input, nil
}

func responseInputItemsForMessage(message provider.Message) ([]responses.ResponseInputItemUnionParam, error) {
	role := strings.ToLower(strings.TrimSpace(message.Role))

	switch role {
	case "assistant":
		return responseAssistantItems(message)
	case "tool":
		if strings.TrimSpace(message.ToolCallID) == "" {
			return nil, fmt.Errorf("agent: tool message requires tool call id")
		}

		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfFunctionCallOutput(message.ToolCallID, message.Content),
		}, nil
	case "system":
		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(message.Content, responses.EasyInputMessageRoleSystem),
		}, nil
	case "user", "":
		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(message.Content, responses.EasyInputMessageRoleUser),
		}, nil
	default:
		return nil, fmt.Errorf("agent: unsupported message role %q", message.Role)
	}
}

func responseAssistantItems(message provider.Message) ([]responses.ResponseInputItemUnionParam, error) {
	items := make([]responses.ResponseInputItemUnionParam, 0, len(message.ToolCalls)+1)

	for _, toolCall := range message.ToolCalls {
		items = append(items, responses.ResponseInputItemParamOfFunctionCall(
			toolCall.Arguments,
			toolCall.ID,
			toolCall.Name,
		))
	}

	if strings.TrimSpace(message.Content) != "" {
		items = append(items, responses.ResponseInputItemParamOfMessage(
			message.Content,
			responses.EasyInputMessageRoleAssistant,
		))
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("agent: assistant message requires content or tool calls")
	}

	return items, nil
}
