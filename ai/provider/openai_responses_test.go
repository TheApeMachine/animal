package provider

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestResponseRequest verifies streaming request construction applies agent parameters.
*/
func TestResponseRequest(t *testing.T) {
	Convey("Given provider params with model controls", t, func() {
		openai := testOpenAIProvider(t)
		agentCtx := NewContext(context.Background())
		So(agentCtx.Append(Message{Role: "user", Content: "hello"}), ShouldBeNil)

		params := NewParams().
			WithModel("override-model").
			WithTemperature(0.7).
			WithTopP(0.9).
			WithMaxOutputTokens(256).
			WithParallelToolCalls(true).
			WithReasoningEffort("low")

		Convey("When responseRequest is built", func() {
			request, err := openai.responseRequest("system", agentCtx, params)

			Convey("Then the request should contain the configured params", func() {
				So(err, ShouldBeNil)
				So(request.Model, ShouldEqual, "override-model")
				So(request.Instructions.Value, ShouldEqual, "system")
				So(request.Temperature.Valid(), ShouldBeTrue)
				So(request.Temperature.Value, ShouldEqual, 0.7)
				So(request.TopP.Value, ShouldEqual, 0.9)
				So(request.MaxOutputTokens.Value, ShouldEqual, 256)
				So(request.ParallelToolCalls.Value, ShouldBeTrue)
				So(string(request.Reasoning.Effort), ShouldEqual, "low")
			})
		})
	})
}

/*
TestParamsValidate verifies bad sampling controls are rejected.
*/
func TestParamsValidate(t *testing.T) {
	Convey("Given an invalid temperature", t, func() {
		params := NewParams().WithTemperature(3)

		Convey("When Validate is called", func() {
			err := params.Validate()

			Convey("Then validation should reject it", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an invalid reasoning effort", t, func() {
		params := NewParams().WithReasoningEffort("wild")

		Convey("When Validate is called", func() {
			err := params.Validate()

			Convey("Then validation should reject it", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
