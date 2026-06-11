package agent

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestParseHeaders verifies MCP session header parsing.
*/
func TestParseHeaders(t *testing.T) {
	Convey("Given agent headers on the request", t, func() {
		header := http.Header{}
		header.Set("X-Agent-ID", "builder-a")
		header.Set("X-Agent-Read-Only", "true")
		header.Set("X-Agent-Require-Lease", "true")
		header.Set("X-Agent-Lease-Prefixes", "lanes/a/, lanes/b/")

		Convey("When ParseHeaders is called", func() {
			access := ParseHeaders(header)

			Convey("Then it should populate access from the headers", func() {
				So(access.ID, ShouldEqual, "builder-a")
				So(access.ReadOnly, ShouldBeTrue)
				So(access.RequireLease, ShouldBeTrue)
				So(len(access.LeasePrefixes), ShouldEqual, 2)
			})
		})
	})

	Convey("Given nil headers", t, func() {
		Convey("When ParseHeaders is called", func() {
			access := ParseHeaders(nil)

			Convey("Then it should return default access", func() {
				So(access.ID, ShouldEqual, DefaultAccess().ID)
				So(access.ReadOnly, ShouldBeFalse)
				So(access.RequireLease, ShouldBeFalse)
			})
		})
	})
}

/*
TestDefaultAccess verifies solo-workflow defaults.
*/
func TestDefaultAccess(t *testing.T) {
	Convey("Given no configured agent headers", t, func() {
		Convey("When DefaultAccess is called", func() {
			access := DefaultAccess()

			Convey("Then it should return the default actor ID", func() {
				So(access.ID, ShouldEqual, "default")
				So(access.ReadOnly, ShouldBeFalse)
				So(access.RequireLease, ShouldBeFalse)
			})
		})
	})
}
