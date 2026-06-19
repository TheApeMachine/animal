package provider

import (
	"context"
	"testing"

	"github.com/openai/openai-go/v3/responses"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/qpool"
)

func testOpenAIProvider(test *testing.T) *OpenAI {
	test.Helper()

	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	openai, err := NewOpenAI(ctx, pool, "http://localhost:1234/v1", "test-key", "test-model")
	So(err, ShouldBeNil)

	return openai
}

/*
TestResponseInputItems verifies role mapping for Responses API requests.
*/
func TestResponseInputItems(t *testing.T) {
	Convey("Given a user message", t, func() {
		openai := testOpenAIProvider(t)

		items, err := openai.responseInputItems([]Message{
			{Role: "user", Content: "Acknowledge the roadmap."},
		})

		Convey("Then responseInputItems should return one user item", func() {
			So(err, ShouldBeNil)
			So(len(items), ShouldEqual, 1)
			So(items[0].OfMessage, ShouldNotBeNil)
			So(items[0].OfMessage.Role, ShouldEqual, responses.EasyInputMessageRoleUser)
		})
	})

	Convey("Given a tool message", t, func() {
		openai := testOpenAIProvider(t)

		items, err := openai.responseInputItems([]Message{
			{
				Role:       "tool",
				ToolCallID: "call_1",
				Content:    `{"url":"https://example.com"}`,
			},
		})

		Convey("Then responseInputItems should map it to function_call_output", func() {
			So(err, ShouldBeNil)
			So(len(items), ShouldEqual, 1)
			So(items[0].OfFunctionCallOutput, ShouldNotBeNil)
			So(items[0].OfFunctionCallOutput.CallID, ShouldEqual, "call_1")
		})
	})

	Convey("Given an assistant message with tool calls", t, func() {
		openai := testOpenAIProvider(t)

		items, err := openai.responseInputItems([]Message{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "lookup", Arguments: `{"q":"leases"}`},
				},
			},
		})

		Convey("Then responseInputItems should emit function_call items", func() {
			So(err, ShouldBeNil)
			So(len(items), ShouldEqual, 1)
			So(items[0].OfFunctionCall, ShouldNotBeNil)
			So(items[0].OfFunctionCall.Name, ShouldEqual, "lookup")
		})
	})
}

/*
TestResponseTextConfig verifies text.format json_schema wiring for structured outputs.
*/
func TestResponseTextConfig(t *testing.T) {
	Convey("Given a structured output definition", t, func() {
		openai := testOpenAIProvider(t)

		textConfig := openai.textConfig(StructuredOutput{
			Name:        "person",
			Description: "A person record",
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "minLength": 1},
					"age":  map[string]any{"type": "number", "minimum": 0, "maximum": 130},
				},
				"required":             []any{"name", "age"},
				"additionalProperties": false,
			},
			Strict: false,
		})

		Convey("Then text.format should use json_schema", func() {
			So(textConfig.Format.OfJSONSchema, ShouldNotBeNil)
			So(textConfig.Format.OfJSONSchema.Name, ShouldEqual, "person")
			So(textConfig.Format.OfJSONSchema.Description.Value, ShouldEqual, "A person record")
			So(textConfig.Format.OfJSONSchema.Strict.Valid(), ShouldBeFalse)
			So(textConfig.Format.OfJSONSchema.Schema["type"], ShouldEqual, "object")
		})
	})
}

/*
TestStructuredOutputValidate verifies required fields.
*/
func TestStructuredOutputValidate(t *testing.T) {
	Convey("Given an incomplete structured output", t, func() {
		err := StructuredOutput{}.Validate()

		Convey("Then Validate should reject it", func() {
			So(err, ShouldNotBeNil)
		})
	})
}

/*
TestJSONSchemaParam verifies chat response_format json_schema wiring.
*/
func TestJSONSchemaParam(t *testing.T) {
	Convey("Given a structured output definition", t, func() {
		schema := jsonSchemaParam(StructuredOutput{
			Name:        "coding_plan",
			Description: "Atomic replace slice",
			Schema: map[string]any{
				"type": "object",
			},
			Strict: false,
		})

		Convey("Then jsonSchemaParam should populate schema fields", func() {
			So(schema.Name, ShouldEqual, "coding_plan")
			So(schema.Description.Value, ShouldEqual, "Atomic replace slice")
			So(schema.Strict.Valid(), ShouldBeFalse)
			So(schema.Schema, ShouldNotBeNil)
		})
	})
}

/*
TestValidateStructuredJSON verifies non-JSON endpoint responses are rejected.
*/
func TestValidateStructuredJSON(t *testing.T) {
	Convey("Given markdown instead of JSON", t, func() {
		err := validateStructuredJSON("```json\n{}\n```")

		Convey("Then validateStructuredJSON should reject it", func() {
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Given valid JSON", t, func() {
		err := validateStructuredJSON(`{"goal_met":true}`)

		Convey("Then validateStructuredJSON should accept it", func() {
			So(err, ShouldBeNil)
		})
	})
}

/*
TestStreamWithSink verifies stream sink validation before network work starts.
*/
func TestStreamWithSink(t *testing.T) {
	Convey("Given a nil stream sink", t, func() {
		openai := testOpenAIProvider(t)

		Convey("When StreamWithSink is called", func() {
			err := openai.StreamWithSink("", NewContext(context.Background()), NewParams(), nil)

			Convey("Then it should reject the call", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
