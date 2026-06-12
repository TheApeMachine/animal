package provider

/*
Message is one turn in the provider conversation log sent to the Responses API.
It carries optional tool-call metadata so assistant and tool-result roles round-trip without lossy conversion.
*/
type Message struct {
	ID         string
	Name       string
	Role       string
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}
