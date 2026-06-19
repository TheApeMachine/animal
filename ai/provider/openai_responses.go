package provider

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"github.com/theapemachine/errnie"
)

func (openai *OpenAI) responseRequest(
	system string,
	agentCtx *Context,
	params *Params,
) (responses.ResponseNewParams, error) {
	if agentCtx == nil {
		return responses.ResponseNewParams{}, errnie.Err(
			errnie.Validation,
			"provider stream requires agent context",
			nil,
		)
	}

	if params == nil {
		return responses.ResponseNewParams{}, errnie.Err(
			errnie.Validation,
			"provider stream requires params",
			nil,
		)
	}

	if err := params.Validate(); err != nil {
		return responses.ResponseNewParams{}, err
	}

	inputItems, err := openai.responseInputItems(agentCtx.Messages)
	if err != nil {
		return responses.ResponseNewParams{}, errnie.Err(
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

	openai.applySystem(&request, system)
	openai.applyParams(&request, params)

	return request, nil
}

func (openai *OpenAI) applySystem(
	request *responses.ResponseNewParams,
	system string,
) {
	if strings.TrimSpace(system) == "" {
		return
	}

	request.Instructions = param.NewOpt(system)
}

func (openai *OpenAI) applyParams(
	request *responses.ResponseNewParams,
	params *Params,
) {
	if strings.TrimSpace(params.Model) != "" {
		request.Model = params.Model
	}

	if params.Temperature != nil {
		request.Temperature = param.NewOpt(*params.Temperature)
	}

	if params.TopP != nil {
		request.TopP = param.NewOpt(*params.TopP)
	}

	if params.MaxOutputTokens != nil {
		request.MaxOutputTokens = param.NewOpt(*params.MaxOutputTokens)
	}

	if params.ParallelToolCalls != nil {
		request.ParallelToolCalls = param.NewOpt(*params.ParallelToolCalls)
	}

	if strings.TrimSpace(params.ReasoningEffort) != "" {
		request.Reasoning = shared.ReasoningParam{
			Effort: shared.ReasoningEffort(params.ReasoningEffort),
		}
	}

	if params.StructuredOutput != nil {
		request.Text = openai.textConfig(*params.StructuredOutput)
	}
}

func (openai *OpenAI) responseInputItems(
	agentMessages []Message,
) (responses.ResponseInputParam, error) {
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
		return openai.toolItems(message)
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

func (openai *OpenAI) toolItems(
	message Message,
) ([]responses.ResponseInputItemUnionParam, error) {
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
}

func (openai *OpenAI) assistantItems(
	message Message,
) ([]responses.ResponseInputItemUnionParam, error) {
	items := make(
		[]responses.ResponseInputItemUnionParam,
		0,
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

func (openai *OpenAI) textConfig(
	structured StructuredOutput,
) responses.ResponseTextConfigParam {
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
