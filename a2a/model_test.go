package a2a

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestTaskValidate verifies A2A task validation and instruction extraction.
*/
func TestTaskValidate(t *testing.T) {
	Convey("Given an A2A task with user history", t, func() {
		task := Task{
			ID: "task-1",
			Status: TaskStatus{
				State: TaskStateSubmitted,
			},
			History: []Message{
				{
					Role: RoleUser,
					Parts: []Part{
						{Text: "Investigate the lease boundary."},
					},
				},
			},
		}

		Convey("When Validate is called", func() {
			err := task.Validate()

			Convey("Then the task should be accepted", func() {
				So(err, ShouldBeNil)
				So(task.Instruction(), ShouldEqual, "Investigate the lease boundary.")
			})
		})
	})

	Convey("Given an A2A task without an ID", t, func() {
		task := Task{
			Status: TaskStatus{
				State: TaskStateSubmitted,
			},
		}

		Convey("When Validate is called", func() {
			err := task.Validate()

			Convey("Then validation should reject it", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestPartValidate verifies member-presence discrimination.
*/
func TestPartValidate(t *testing.T) {
	Convey("Given a text part", t, func() {
		part := Part{Text: "status"}

		Convey("When Validate is called", func() {
			err := part.Validate()

			Convey("Then it should be accepted", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a part with two payloads", t, func() {
		part := Part{
			Text: "status",
			Data: map[string]any{"ok": true},
		}

		Convey("When Validate is called", func() {
			err := part.Validate()

			Convey("Then it should be rejected", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

/*
TestTaskStatusUpdateEventValidate verifies streaming event validation.
*/
func TestTaskStatusUpdateEventValidate(t *testing.T) {
	Convey("Given a task status update event", t, func() {
		event := TaskStatusUpdateEvent{
			TaskID: "task-1",
			Status: TaskStatus{
				State: TaskStateWorking,
			},
		}

		Convey("When Validate is called", func() {
			err := event.Validate()

			Convey("Then it should be accepted", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

/*
TestAgentCardValidate verifies A2A discovery card JSON shape.
*/
func TestAgentCardValidate(t *testing.T) {
	Convey("Given an A2A agent card with streaming capability", t, func() {
		card := AgentCard{
			Name: "animal-agent",
			URL:  "http://localhost:8080/a2a",
			Capabilities: AgentCapabilities{
				Streaming: true,
			},
			ProtocolVersions: []string{"1.0"},
		}

		Convey("When the card is validated and marshaled", func() {
			err := card.Validate()
			payload, marshalErr := json.Marshal(card)

			Convey("Then it should be valid JSON with capabilities", func() {
				So(err, ShouldBeNil)
				So(marshalErr, ShouldBeNil)
				So(string(payload), ShouldContainSubstring, `"streaming":true`)
				So(string(payload), ShouldContainSubstring, `"protocolVersions":["1.0"]`)
			})
		})
	})
}
