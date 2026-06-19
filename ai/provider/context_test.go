package provider

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestContextClone verifies message history is copied without slice sharing.
*/
func TestContextClone(t *testing.T) {
	Convey("Given a provider context with one message", t, func() {
		agentCtx := NewContext(context.Background())
		So(agentCtx.Append(Message{Role: "user", Content: "initial"}), ShouldBeNil)

		Convey("When Clone is called and the clone is changed", func() {
			clone, err := agentCtx.Clone(context.Background())
			So(err, ShouldBeNil)
			So(clone.Append(Message{Role: "user", Content: "clone task"}), ShouldBeNil)

			Convey("Then the original history should remain unchanged", func() {
				So(len(agentCtx.Messages), ShouldEqual, 1)
				So(len(clone.Messages), ShouldEqual, 2)
				So(clone.Messages[1].Content, ShouldEqual, "clone task")
			})
		})
	})
}

/*
TestContextReplace verifies full message history replacement.
*/
func TestContextReplace(t *testing.T) {
	Convey("Given a provider context", t, func() {
		agentCtx := NewContext(context.Background())

		Convey("When Replace is called", func() {
			err := agentCtx.Replace([]Message{
				{Role: "user", Content: "redirected"},
			})

			Convey("Then the context should expose the replacement history", func() {
				So(err, ShouldBeNil)
				So(len(agentCtx.Messages), ShouldEqual, 1)
				So(agentCtx.Messages[0].Content, ShouldEqual, "redirected")
			})
		})
	})
}
