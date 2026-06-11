package provider

import "encoding/json"

/*
ToolDefinition describes one function tool exposed to the model.
*/
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

/*
ToolCall is one function invocation requested by the model.
*/
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

/*
ToolCompletion is an assistant turn that may include tool calls.
*/
type ToolCompletion struct {
	Content   string
	ToolCalls []ToolCall
}
