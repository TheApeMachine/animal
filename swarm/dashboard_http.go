package swarm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/theapemachine/errnie"
)

/*
ServeHTTP routes dashboard HTML, snapshot JSON, and live SSE updates.
*/
func (dashboard *Dashboard) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	switch request.URL.Path {
	case "/", "/dashboard":
		dashboard.handleIndex(response)
	case "/snapshot", "/dashboard/snapshot":
		dashboard.handleSnapshot(response)
	case "/events", "/dashboard/events":
		dashboard.handleEvents(response, request)
	default:
		http.NotFound(response, request)
	}
}

func (dashboard *Dashboard) handleIndex(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	dashboard.write(response, []byte(dashboardHTML))
}

func (dashboard *Dashboard) handleSnapshot(response http.ResponseWriter) {
	data, err := json.MarshalIndent(dashboard.Snapshot(), "", "  ")

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)

		return
	}

	response.Header().Set("Content-Type", "application/json")
	dashboard.write(response, data)
}

func (dashboard *Dashboard) handleEvents(
	response http.ResponseWriter,
	request *http.Request,
) {
	flusher, ok := response.(http.Flusher)

	if !ok {
		http.Error(response, "streaming unsupported", http.StatusInternalServerError)

		return
	}

	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("Connection", "keep-alive")

	if err := dashboard.writeEvent(response); err != nil {
		errnie.Error(err)

		return
	}

	flusher.Flush()
	dashboard.streamEvents(response, request, flusher)
}

func (dashboard *Dashboard) streamEvents(
	response http.ResponseWriter,
	request *http.Request,
	flusher http.Flusher,
) {
	ticker := time.NewTicker(dashboard.refresh)
	defer ticker.Stop()

	for {
		select {
		case <-dashboard.ctx.Done():
			return
		case <-request.Context().Done():
			return
		case <-ticker.C:
			if err := dashboard.writeEvent(response); err != nil {
				errnie.Error(err)

				return
			}

			flusher.Flush()
		}
	}
}

func (dashboard *Dashboard) writeEvent(response http.ResponseWriter) error {
	data, err := json.Marshal(dashboard.Snapshot())

	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(response, "event: snapshot\ndata: %s\n\n", data)

	return err
}

func (dashboard *Dashboard) write(response http.ResponseWriter, data []byte) {
	_, err := response.Write(data)

	if err != nil {
		errnie.Error(err)
	}
}
