package provider

type Message struct {
	ID         string
	Name       string
	Role       string
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}
