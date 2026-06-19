package provider

import (
	"context"
	"fmt"
	"strings"
)

/*
Params bundles model, message, and structured-output settings for one Responses API call.
The fluent With* methods let callers reuse defaults while overriding only the fields a workflow step needs.
*/
type Params struct {
	Context           *Context
	Model             string
	Messages          []Message
	Strict            bool
	Format            string
	StructuredOutput  *StructuredOutput
	Temperature       *float64
	TopP              *float64
	MaxOutputTokens   *int64
	ParallelToolCalls *bool
	ReasoningEffort   string
}

func NewParams() *Params {
	return &Params{
		Context:          NewContext(context.Background()),
		Model:            "",
		Messages:         make([]Message, 0),
		Strict:           false,
		Format:           "json_schema",
		StructuredOutput: nil,
	}
}

func (params *Params) WithContext(ctx *Context) *Params {
	params.Context = ctx
	return params
}

func (params *Params) WithModel(model string) *Params {
	params.Model = model
	return params
}

func (params *Params) WithMessages(messages []Message) *Params {
	params.Messages = messages
	return params
}

func (params *Params) WithStrict(strict bool) *Params {
	params.Strict = strict
	return params
}

func (params *Params) WithFormat(format string) *Params {
	params.Format = format
	return params
}

func (params *Params) WithStructuredOutput(structured *StructuredOutput) *Params {
	params.StructuredOutput = structured
	return params
}

func (params *Params) WithTemperature(temperature float64) *Params {
	params.Temperature = &temperature
	return params
}

func (params *Params) WithTopP(topP float64) *Params {
	params.TopP = &topP
	return params
}

func (params *Params) WithMaxOutputTokens(maxOutputTokens int64) *Params {
	params.MaxOutputTokens = &maxOutputTokens
	return params
}

func (params *Params) WithParallelToolCalls(parallelToolCalls bool) *Params {
	params.ParallelToolCalls = &parallelToolCalls
	return params
}

func (params *Params) WithReasoningEffort(reasoningEffort string) *Params {
	params.ReasoningEffort = reasoningEffort
	return params
}

func (params *Params) Validate() error {
	if params.Temperature != nil && (*params.Temperature < 0 || *params.Temperature > 2) {
		return fmt.Errorf("provider: temperature must be between 0 and 2")
	}

	if params.TopP != nil && (*params.TopP < 0 || *params.TopP > 1) {
		return fmt.Errorf("provider: top_p must be between 0 and 1")
	}

	if params.MaxOutputTokens != nil && *params.MaxOutputTokens <= 0 {
		return fmt.Errorf("provider: max output tokens must be positive")
	}

	if strings.TrimSpace(params.ReasoningEffort) == "" {
		return params.validateStructuredOutput()
	}

	if !validReasoningEffort(params.ReasoningEffort) {
		return fmt.Errorf("provider: unsupported reasoning effort %q", params.ReasoningEffort)
	}

	return params.validateStructuredOutput()
}

func (params *Params) validateStructuredOutput() error {
	if params.StructuredOutput == nil {
		return nil
	}

	return params.StructuredOutput.Validate()
}

func validReasoningEffort(reasoningEffort string) bool {
	switch strings.TrimSpace(reasoningEffort) {
	case "none", "minimal", "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

/*
StructuredOutput configures Responses API text.format json_schema output.

See https://platform.openai.com/docs/guides/structured-outputs
*/
type StructuredOutput struct {
	Name        string
	Description string
	Schema      map[string]any
	Strict      bool
}

/*
Validate checks required structured output fields.
*/
func (structured StructuredOutput) Validate() error {
	if strings.TrimSpace(structured.Name) == "" {
		return fmt.Errorf("provider: structured output name is required")
	}

	if len(structured.Schema) == 0 {
		return fmt.Errorf("provider: structured output schema is required")
	}

	return nil
}
