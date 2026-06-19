package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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
	return openai.StreamWithSink(system, agentCtx, params, func(delta string) error {
		return openai.bus.Send(internal.ChannelMessages, "text", delta)
	})
}

/*
StreamWithSink sends response text deltas to sink until the model finishes.
*/
func (openai *OpenAI) StreamWithSink(
	system string,
	agentCtx *Context,
	params *Params,
	sink func(string) error,
) error {
	if sink == nil {
		return errnie.Err(errnie.Validation, "provider stream sink is required", nil)
	}

	request, err := openai.responseRequest(system, agentCtx, params)
	if err != nil {
		return err
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

		if deltaEvent.Delta == "" {
			continue
		}

		if err := sink(deltaEvent.Delta); err != nil {
			return errnie.Err(
				errnie.IO,
				"provider stream sink failed",
				err,
			)
		}
	}

	return errnie.Error(stream.Err())
}
