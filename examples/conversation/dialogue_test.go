package conversation

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestBuildReplyMessages verifies transcript roles and a persistent moderator anchor.
*/
func TestBuildReplyMessages(t *testing.T) {
	Convey("Given prior turns in the salon", t, func() {
		turns := []Turn{
			{ActorName: "Elena", Content: "We should define sentience first."},
			{ActorName: "Sam", Content: "I doubt the premise."},
			{ActorName: "Elena", Content: "Fair, but the question still stands."},
		}

		Convey("When BuildReplyMessages is called for Elena", func() {
			messages := BuildReplyMessages(turns, "What should humanity do?", "Elena")

			Convey("Then it should anchor the moderator and separate self from others", func() {
				So(len(messages), ShouldEqual, 4)
				So(messages[0].Content, ShouldContainSubstring, "Moderator:")
				So(messages[1].Role, ShouldEqual, "assistant")
				So(messages[2].Role, ShouldEqual, "user")
				So(messages[2].Content, ShouldContainSubstring, "Sam:")
				So(messages[3].Role, ShouldEqual, "assistant")
			})
		})
	})
}

/*
TestThemesFromContent verifies stance tags are inferred from speech.
*/
func TestThemesFromContent(t *testing.T) {
	Convey("Given speech about governance and rights", t, func() {
		themes := ThemesFromContent("We need transparent governance and moral rights.")

		Convey("Then ThemesFromContent should tag matching themes", func() {
			So(themes, ShouldContain, "governance")
			So(themes, ShouldContain, "rights")
		})
	})
}
