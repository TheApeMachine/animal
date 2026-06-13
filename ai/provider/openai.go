package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/theapemachine/animal/internal"
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
	bus      *internal.Bus
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
		ctx:    ctx,
		cancel: cancel,
		pool:   pool,
		bus: internal.NewBus(
			ctx,
			pool,
			[]internal.Channel{internal.ChannelMessages},
			[]internal.Subscription{
				internal.Subscribe(internal.ChannelMessages, "openai"),
			}),
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
		"bus":      openai.bus,
		"endpoint": openai.endpoint,
		"apiKey":   openai.apiKey,
		"model":    openai.model,
		"client":   openai.client,
	})
}

/*
CompleteStructured requests schema-bound JSON through chat response_format json_schema.
Local OpenAI-compatible servers typically implement this path; the Responses API text.format
path is reserved for Stream and other provider flows.
*/
func (openai *OpenAI) CompleteStructured(
	ctx context.Context,
	system string,
	user string,
	structured StructuredOutput,
) ([]byte, error) {
	if err := structured.Validate(); err != nil {
		return nil, err
	}

	messages := make([]openaiapi.ChatCompletionMessageParamUnion, 0, 2)

	if strings.TrimSpace(system) != "" {
		messages = append(messages, openaiapi.ChatCompletionMessageParamUnion{
			OfSystem: &openaiapi.ChatCompletionSystemMessageParam{
				Content: openaiapi.ChatCompletionSystemMessageParamContentUnion{
					OfString: openaiapi.String(system),
				},
			},
		})
	}

	messages = append(messages, openaiapi.ChatCompletionMessageParamUnion{
		OfUser: &openaiapi.ChatCompletionUserMessageParam{
			Content: openaiapi.ChatCompletionUserMessageParamContentUnion{
				OfString: openaiapi.String(user),
			},
		},
	})

	completion, err := openai.client.Chat.Completions.New(ctx, openaiapi.ChatCompletionNewParams{
		Model:    openai.model,
		Messages: messages,
		ResponseFormat: openaiapi.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: jsonSchemaParam(structured),
			},
		},
	})

	if err != nil {
		return nil, errnie.Err(
			errnie.IO,
			"provider structured completion failed",
			err,
		)
	}

	if len(completion.Choices) == 0 {
		return nil, errnie.Err(
			errnie.IO,
			"provider structured completion returned no choices",
			nil,
		)
	}

	payload := strings.TrimSpace(completion.Choices[0].Message.Content)

	if payload == "" {
		return nil, errnie.Err(
			errnie.IO,
			"provider structured completion returned empty content",
			nil,
		)
	}

	if err := validateStructuredJSON(payload); err != nil {
		return nil, err
	}

	return []byte(payload), nil
}

func jsonSchemaParam(structured StructuredOutput) shared.ResponseFormatJSONSchemaJSONSchemaParam {
	schema := shared.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:   structured.Name,
		Schema: structured.Schema,
	}

	if structured.Strict {
		schema.Strict = openaiapi.Bool(true)
	}

	if strings.TrimSpace(structured.Description) != "" {
		schema.Description = openaiapi.String(structured.Description)
	}

	return schema
}

func validateStructuredJSON(payload string) error {
	trimmed := strings.TrimSpace(payload)

	if json.Valid([]byte(trimmed)) {
		return nil
	}

	preview := trimmed
	if len(preview) > 120 {
		preview = preview[:120]
	}

	return errnie.Err(
		errnie.IO,
		fmt.Sprintf(
			"provider endpoint returned non-JSON (%q)",
			preview,
		),
		nil,
	)
}

/*
Stream sends response text deltas to broadcast until the model finishes.
*/
func (openai *OpenAI) Stream(
	system string, agentCtx *Context, params *Params,
) error {
	if agentCtx == nil {
		return errnie.Err(
			errnie.Validation,
			"provider stream requires agent context",
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
		if stream.Err() != nil {
			return errnie.Err(
				errnie.IO,
				"provider stream next failed",
				stream.Err(),
			)
		}

		event := stream.Current()
		deltaEvent := event.AsResponseOutputTextDelta()

		if strings.TrimSpace(deltaEvent.Delta) == "" {
			continue
		}

		qv, err := qpool.NewQValue[any]("", "", deltaEvent.Delta, time.Minute)

		if err != nil {
			return errnie.Err(
				errnie.IO,
				"provider stream qvalue failed",
				err,
			)
		}

		if err := openai.bus.Send(internal.ChannelMessages, "text", qv.Value); err != nil {
			return errnie.Err(
				errnie.IO,
				"provider stream bus send failed",
				errnie.Err(
					errnie.Validation,
					"provider stream bus send failed",
					err,
				),
			)
		}
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

	schema := jsonSchemaParam(structured)

	if schema.Strict.Valid() && format.OfJSONSchema != nil {
		format.OfJSONSchema.Strict = schema.Strict
	}

	if schema.Description.Valid() && format.OfJSONSchema != nil {
		format.OfJSONSchema.Description = schema.Description
	}

	return responses.ResponseTextConfigParam{
		Format: format,
	}
}
