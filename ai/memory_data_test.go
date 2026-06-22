package ai

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura/types"
)

func TestMemoryPacketMerge(t *testing.T) {
	Convey("Given a memory packet and scoped datura result", t, func() {
		packet := MemoryPacket{}
		result := types.Memory{
			Documents: []types.Document{
				{
					ID:   "doc-1",
					Text: "Use recall before generation.",
					Metadata: types.Metadata{
						Source: "goal-a",
					},
				},
				{
					ID:   "doc-2",
					Text: "Skip unrelated memory.",
					Metadata: types.Metadata{
						Source: "goal-b",
					},
				},
			},
		}

		packet.merge(result, "goal-a")

		Convey("It should keep only memories for that scope", func() {
			So(packet.Documents, ShouldHaveLength, 1)
			So(packet.Documents[0].ID, ShouldEqual, "doc-1")
		})
	})

	Convey("Given repeated memories across query results", t, func() {
		packet := MemoryPacket{}
		result := types.Memory{
			Documents: []types.Document{
				{
					ID:   "doc-1",
					Text: "Use recall before generation.",
					Metadata: types.Metadata{
						Source: "goal-a",
					},
				},
			},
			Relationships: []types.Relationship{
				{
					ID:           "doc-1",
					Relationship: "supports",
					ToID:         "doc-2",
					Metadata: types.Metadata{
						ID:     "rel-1",
						Source: "goal-a",
					},
				},
			},
		}

		packet.merge(result, "goal-a")
		packet.merge(result, "goal-a")

		Convey("It should keep one copy of each memory", func() {
			So(packet.Documents, ShouldHaveLength, 1)
			So(packet.Relationships, ShouldHaveLength, 1)
		})
	})
}

func TestMemoryQueryDatura(t *testing.T) {
	Convey("Given a memory query with surrounding whitespace", t, func() {
		query := MemoryQuery{
			ID:           " query-1 ",
			Text:         " swarm memory ",
			Embedding:    []float32{0.25},
			Limit:        3,
			VectorWeight: 0.7,
			TextWeight:   0.3,
		}

		daturaQuery := query.datura()
		daturaQuery.Embedding[0] = 0.75

		Convey("It should normalize scalar fields and copy embeddings", func() {
			So(daturaQuery.ID, ShouldEqual, "query-1")
			So(daturaQuery.Text, ShouldEqual, "swarm memory")
			So(daturaQuery.Embedding, ShouldResemble, []float32{0.75})
			So(query.Embedding, ShouldResemble, []float32{0.25})
		})
	})
}

func TestMemoryRecordMutation(t *testing.T) {
	Convey("Given a memory record", t, func() {
		record := MemoryRecord{
			ID:        " record-1 ",
			Scope:     " goal-a ",
			Text:      " Keep this. ",
			Embedding: []float32{0.5},
		}

		mutation := record.mutation()

		Convey("It should produce a datura mutation", func() {
			So(mutation.ID, ShouldEqual, "record-1")
			So(mutation.Text, ShouldEqual, "Keep this.")
			So(mutation.Metadata.ID, ShouldEqual, "record-1")
			So(mutation.Metadata.Source, ShouldEqual, "goal-a")
			So(mutation.Embedding, ShouldResemble, []float32{0.5})
		})
	})
}

func TestMemoryRelationshipMutation(t *testing.T) {
	Convey("Given a memory relationship", t, func() {
		relationship := MemoryRelationship{
			ID:           " edge-1 ",
			Scope:        " goal-a ",
			FromID:       " doc-1 ",
			ToID:         " doc-2 ",
			Relationship: " supports ",
		}

		mutation := relationship.mutation()

		Convey("It should produce a graph mutation", func() {
			So(mutation.ID, ShouldEqual, "doc-1")
			So(mutation.RelatedID, ShouldEqual, "doc-2")
			So(mutation.Relationship, ShouldEqual, "supports")
			So(mutation.Metadata.ID, ShouldEqual, "edge-1")
			So(mutation.Metadata.Source, ShouldEqual, "goal-a")
		})
	})
}

func TestMemoryPacketFormat(t *testing.T) {
	Convey("Given an empty memory packet", t, func() {
		packet := MemoryPacket{}

		Convey("It should format to an empty string", func() {
			So(packet.Format(), ShouldEqual, "")
		})
	})

	Convey("Given documents and relationships", t, func() {
		packet := MemoryPacket{
			Documents: []MemoryDocument{
				{
					ID:    "doc-1",
					Scope: "goal-a",
					Text:  "Prefer recall before the task response.",
				},
			},
			Relationships: []MemoryRelationship{
				{
					ID:           "rel-1",
					Scope:        "goal-a",
					FromID:       "doc-1",
					ToID:         "doc-2",
					Relationship: "supports",
				},
			},
		}

		formatted := packet.Format()

		Convey("It should render compact injectable memory", func() {
			So(formatted, ShouldContainSubstring, "Relevant memory:")
			So(formatted, ShouldContainSubstring, "[goal-a/doc-1]")
			So(formatted, ShouldContainSubstring, "doc-1 -supports-> doc-2")
		})
	})
}
