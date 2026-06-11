package provider

import (
	"context"
	"fmt"
	"strings"
)

type Params struct {
	Context          *Context
	Model            string
	Messages         []Message
	Strict           bool
	Format           string
	StructuredOutput *StructuredOutput
}

func NewParams() *Params {
	return &Params{
		Context:          NewContext(context.Background()),
		Model:            "gpt-5.5-mini",
		Messages:         make([]Message, 0),
		Strict:           true,
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
