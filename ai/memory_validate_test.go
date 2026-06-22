package ai

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMemoryRecallPlanValidate(t *testing.T) {
	Convey("Given a recall plan with an invalid query", t, func() {
		plan := MemoryRecallPlan{
			Queries: []MemoryQuery{{Text: "   ", Limit: 3}},
		}

		Convey("It should return validation error", func() {
			So(plan.Validate(), ShouldNotBeNil)
		})
	})
}

func TestMemoryQueryValidate(t *testing.T) {
	Convey("Given a memory query with text and limit", t, func() {
		query := MemoryQuery{Text: "goal", Limit: 3}

		Convey("It should pass validation", func() {
			So(query.Validate(), ShouldBeNil)
		})
	})

	Convey("Given a memory query without a limit", t, func() {
		query := MemoryQuery{Text: "goal"}

		Convey("It should return validation error", func() {
			So(query.Validate(), ShouldNotBeNil)
		})
	})
}

func TestMemoryConsolidationValidate(t *testing.T) {
	Convey("Given valid consolidation output", t, func() {
		consolidation := MemoryConsolidation{
			Records: []MemoryRecord{
				{
					ID:         "record-1",
					Scope:      "goal-a",
					Text:       "Remember this.",
					Importance: 0.8,
				},
			},
			Relationships: []MemoryRelationship{
				{
					ID:           "edge-1",
					Scope:        "goal-a",
					FromID:       "record-1",
					ToID:         "record-2",
					Relationship: "supports",
					Importance:   0.6,
				},
			},
		}

		Convey("It should pass validation", func() {
			So(consolidation.Validate(), ShouldBeNil)
		})
	})
}

func TestMemoryRecordValidate(t *testing.T) {
	Convey("Given a memory record without scope", t, func() {
		record := MemoryRecord{
			ID:         "record-1",
			Text:       "Remember this.",
			Importance: 0.5,
		}

		Convey("It should return validation error", func() {
			So(record.Validate(), ShouldNotBeNil)
		})
	})

	Convey("Given a memory record with invalid importance", t, func() {
		record := MemoryRecord{
			ID:         "record-1",
			Scope:      "goal-a",
			Text:       "Remember this.",
			Importance: 1.5,
		}

		Convey("It should return validation error", func() {
			So(record.Validate(), ShouldNotBeNil)
		})
	})
}

func TestMemoryRelationshipValidate(t *testing.T) {
	Convey("Given a memory relationship without target", t, func() {
		relationship := MemoryRelationship{
			ID:           "edge-1",
			Scope:        "goal-a",
			FromID:       "record-1",
			Relationship: "supports",
			Importance:   0.5,
		}

		Convey("It should return validation error", func() {
			So(relationship.Validate(), ShouldNotBeNil)
		})
	})

	Convey("Given a memory relationship with invalid importance", t, func() {
		relationship := MemoryRelationship{
			ID:           "edge-1",
			Scope:        "goal-a",
			FromID:       "record-1",
			ToID:         "record-2",
			Relationship: "supports",
			Importance:   -0.1,
		}

		Convey("It should return validation error", func() {
			So(relationship.Validate(), ShouldNotBeNil)
		})
	})
}
