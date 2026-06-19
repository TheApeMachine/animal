package a2a

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestTaskClone verifies task clones do not share mutable maps and slices.
*/
func TestTaskClone(t *testing.T) {
	Convey("Given a task with nested mutable fields", t, func() {
		task := Task{
			ID: "task-1",
			Status: TaskStatus{
				State: TaskStateSubmitted,
				Message: &Message{
					Role: RoleAgent,
					Parts: []Part{
						{Data: map[string]any{"summary": "ready"}},
					},
					Metadata: map[string]any{"actor_id": "actor-a"},
				},
			},
			History: []Message{
				{
					Role: RoleUser,
					Parts: []Part{
						{Text: "inspect"},
					},
				},
			},
			Metadata: map[string]any{"priority": "high"},
		}

		Convey("When Clone is called and mutated", func() {
			clone := task.Clone()
			clone.Metadata["priority"] = "low"
			clone.Status.Message.Metadata["actor_id"] = "actor-b"
			clone.Status.Message.Parts[0].Data["summary"] = "changed"
			clone.History[0].Parts[0].Text = "mutated"

			Convey("Then the original task should remain unchanged", func() {
				So(task.Metadata["priority"], ShouldEqual, "high")
				So(task.Status.Message.Metadata["actor_id"], ShouldEqual, "actor-a")
				So(task.Status.Message.Parts[0].Data["summary"], ShouldEqual, "ready")
				So(task.History[0].Parts[0].Text, ShouldEqual, "inspect")
			})
		})
	})
}
