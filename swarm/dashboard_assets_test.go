package swarm

import (
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDashboardHTML verifies the embedded dashboard shell points at live endpoints.
*/
func TestDashboardHTML(t *testing.T) {
	Convey("Given the dashboard HTML shell", t, func() {
		Convey("When it is inspected", func() {
			Convey("Then it should wire snapshot and event endpoints", func() {
				So(strings.Contains(dashboardHTML, "new EventSource('/events')"), ShouldBeTrue)
				So(strings.Contains(dashboardHTML, "fetch('/snapshot')"), ShouldBeTrue)
			})
		})
	})
}
