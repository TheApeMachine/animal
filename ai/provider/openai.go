package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
OpenAI streams model output through the Responses API.
*/
type OpenAI struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	pool     *qpool.Q[any]
	endpoint string
	apiKey   string
	model    string
	client   openaiapi.Client
}

/*
NewOpenAI wires a client to endpoint, apiKey, and model.
*/
func NewOpenAI(
	ctx context.Context,
	pool *qpool.Q[any],
	endpoint, apiKey, model string,
) (*OpenAI, error) {
	ctx, cancel := context.WithCancel(ctx)

	openai := &OpenAI{
		ctx:      ctx,
		cancel:   cancel,
		pool:     pool,
		endpoint: strings.TrimRight(endpoint, "/"),
		apiKey:   apiKey,
		model:    model,
	}

	openai.client = openaiapi.NewClient(
		option.WithBaseURL(openai.endpoint),
		option.WithAPIKey(openai.apiKey),
	)

	return openai, errnie.Require(map[string]any{
		"ctx":      openai.ctx,
		"cancel":   openai.cancel,
		"pool":     openai.pool,
		"endpoint": openai.endpoint,
		"apiKey":   openai.apiKey,
		"model":    openai.model,
		"client":   openai.client,
	})
}

/*
Stream sends response text deltas to broadcast until the model finishes.
*/
func (openai *OpenAI) Stream(
	system string,
	agentCtx *Context,
	broadcast *qpool.BroadcastGroup,
	params *Params,
) error {
	if agentCtx == nil {
		return errnie.Err(
			errnie.Validation,
			"provider stream requires agent context",
			nil,
		)
	}

	if broadcast == nil {
		return errnie.Err(
			errnie.Validation,
			"provider stream requires broadcast group",
			nil,
		)
	}

	inputItems, err := openai.responseInputItems(agentCtx.Messages)

	if err != nil {
		return errnie.Err(
			errnie.Validation,
			"provider stream input build failed",
			err,
		)
	}

	request := responses.ResponseNewParams{
		Model: openai.model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
	}

	if strings.TrimSpace(system) != "" {
		request.Instructions = param.NewOpt(system)
	}

	if params.StructuredOutput != nil {
		if err := params.StructuredOutput.Validate(); err != nil {
			return errnie.Err(
				errnie.Validation,
				"provider stream structured output invalid",
				err,
			)
		}

		request.Text = openai.textConfig(*params.StructuredOutput)
	}

	stream := openai.client.Responses.NewStreaming(openai.ctx, request)

	for stream.Next() {
		event := stream.Current()
		deltaEvent := event.AsResponseOutputTextDelta()

		if strings.TrimSpace(deltaEvent.Delta) == "" {
			continue
		}

		qv, qvErr := qpool.NewQValue[any]("", "", deltaEvent.Delta, time.Minute)
		if qvErr != nil {
			return errnie.Err(
				errnie.Validation,
				"provider stream qvalue failed",
				qvErr,
			)
		}

		broadcast.Send(qv)
	}

	return errnie.Error(stream.Err())
}

func (openai *OpenAI) responseInputItems(agentMessages []Message) (responses.ResponseInputParam, error) {
	input := make(responses.ResponseInputParam, 0, len(agentMessages))

	for _, message := range agentMessages {
		items, err := openai.inputItemsForMessage(message)
		if err != nil {
			return nil, errnie.Err(
				errnie.Validation,
				"provider response input items for message failed",
				err,
			)
		}

		input = append(input, items...)
	}

	if len(input) == 0 {
		return nil, errnie.Err(
			errnie.Validation,
			"provider response input items requires at least one message",
			nil,
		)
	}

	return input, nil
}

func (openai *OpenAI) inputItemsForMessage(
	message Message,
) ([]responses.ResponseInputItemUnionParam, error) {
	role := strings.ToLower(strings.TrimSpace(message.Role))

	switch role {
	case "assistant":
		return openai.assistantItems(message)
	case "tool":
		if strings.TrimSpace(message.ToolCallID) == "" {
			return nil, errnie.Err(
				errnie.Validation,
				"provider: tool message requires tool call id",
				nil,
			)
		}

		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfFunctionCallOutput(
				message.ToolCallID,
				message.Content,
			),
		}, nil
	case "system":
		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(
				message.Content,
				responses.EasyInputMessageRoleSystem,
			),
		}, nil
	case "user", "":
		return []responses.ResponseInputItemUnionParam{
			responses.ResponseInputItemParamOfMessage(
				message.Content,
				responses.EasyInputMessageRoleUser,
			),
		}, nil
	default:
		return nil, errnie.Err(
			errnie.Validation,
			fmt.Sprintf("provider: unsupported message role %q", message.Role),
			nil,
		)
	}
}

func (openai *OpenAI) assistantItems(
	message Message,
) ([]responses.ResponseInputItemUnionParam, error) {
	items := make(
		[]responses.ResponseInputItemUnionParam,
		len(message.ToolCalls)+1,
	)

	for _, toolCall := range message.ToolCalls {
		items = append(
			items,
			responses.ResponseInputItemParamOfFunctionCall(
				toolCall.Arguments,
				toolCall.ID,
				toolCall.Name,
			),
		)
	}

	if strings.TrimSpace(message.Content) != "" {
		items = append(items, responses.ResponseInputItemParamOfMessage(
			message.Content,
			responses.EasyInputMessageRoleAssistant,
		))
	}

	if len(items) == 0 {
		return nil, errnie.Err(
			errnie.Validation,
			"provider response assistant items requires content or tool calls",
			nil,
		)
	}

	return items, nil
}

func (openai *OpenAI) textConfig(structured StructuredOutput) responses.ResponseTextConfigParam {
	format := responses.ResponseFormatTextConfigParamOfJSONSchema(
		structured.Name,
		structured.Schema,
	)

	if structured.Strict && format.OfJSONSchema != nil {
		format.OfJSONSchema.Strict = openaiapi.Bool(true)
	}

	if strings.TrimSpace(structured.Description) != "" && format.OfJSONSchema != nil {
		format.OfJSONSchema.Description = openaiapi.String(structured.Description)
	}

	return responses.ResponseTextConfigParam{
		Format: format,
	}
}
