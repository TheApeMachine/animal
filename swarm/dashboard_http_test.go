package swarm

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestDashboardServeHTTP verifies dashboard routing.
*/
func TestDashboardServeHTTP(t *testing.T) {
	Convey("Given a dashboard handler", t, func() {
		dashboard := testDashboard(t)
		defer func() {
			So(dashboard.Close(), ShouldBeNil)
		}()

		Convey("When the index route is requested", func() {
			request := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
			response := httptest.NewRecorder()
			dashboard.ServeHTTP(response, request)

			Convey("Then HTML should be returned", func() {
				So(response.Code, ShouldEqual, http.StatusOK)
				So(response.Body.String(), ShouldContainSubstring, "animal swarm dashboard")
			})
		})
	})
}

/*
TestDashboardHandleSnapshot verifies snapshot JSON responses.
*/
func TestDashboardHandleSnapshot(t *testing.T) {
	Convey("Given a dashboard handler", t, func() {
		dashboard := testDashboard(t)
		defer func() {
			So(dashboard.Close(), ShouldBeNil)
		}()

		Convey("When the snapshot route is requested", func() {
			request := httptest.NewRequest(http.MethodGet, "/snapshot", nil)
			response := httptest.NewRecorder()
			dashboard.ServeHTTP(response, request)

			snapshot := DashboardSnapshot{}
			err := json.Unmarshal(response.Body.Bytes(), &snapshot)

			Convey("Then JSON should describe the current swarm view", func() {
				So(response.Code, ShouldEqual, http.StatusOK)
				So(err, ShouldBeNil)
				So(len(snapshot.Tasks), ShouldEqual, 1)
				So(snapshot.Tasks[0].ID, ShouldEqual, "task-1")
			})
		})
	})
}

/*
TestDashboardHandleEvents verifies SSE emits an initial snapshot.
*/
func TestDashboardHandleEvents(t *testing.T) {
	Convey("Given a dashboard handler", t, func() {
		dashboard := testDashboard(t)
		defer func() {
			So(dashboard.Close(), ShouldBeNil)
		}()

		server := httptest.NewServer(dashboard)
		defer server.Close()

		Convey("When the events route is requested", func() {
			response, err := http.Get(server.URL + "/events")
			So(err, ShouldBeNil)

			if err != nil {
				return
			}

			defer response.Body.Close()

			scanner := bufio.NewScanner(response.Body)
			line := firstSSEDataLine(scanner)

			Convey("Then it should stream snapshot data", func() {
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(line, ShouldStartWith, "data: ")
				So(line, ShouldContainSubstring, `"Tasks"`)
			})
		})
	})
}

func testDashboard(t *testing.T) *Dashboard {
	t.Helper()

	view, err := dashboardView()

	if err != nil {
		t.Fatal(err)
	}

	dashboard, err := NewDashboard(
		t.Context(),
		view,
		DashboardOptions{Refresh: time.Hour},
	)

	if err != nil {
		t.Fatal(err)
	}

	return dashboard
}

func firstSSEDataLine(scanner *bufio.Scanner) string {
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		return line
	}

	return ""
}
