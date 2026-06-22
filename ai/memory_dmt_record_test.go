package ai

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura/types"
	"github.com/theapemachine/errnie"
)

func TestNewDMTDocumentRecord(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a document mutation", t, func() {
		mutation := dmtMemoryMutation("record-1", "goal-record", "Remember cognitive routes.")

		Convey("It should build a document record", func() {
			record, err := newDMTDocumentRecord(mutation)

			So(err, ShouldBeNil)
			So(record.ID, ShouldEqual, "record-1")
			So(record.Scope, ShouldEqual, "goal-record")
			So(record.documentKey(), ShouldContainSubstring, "goal_record")
		})
	})

	Convey("Given a mutation without cognitive text", t, func() {
		mutation := dmtMemoryMutation("record-2", "goal-record", "   ...   ")

		Convey("It should return validation error", func() {
			record, err := newDMTDocumentRecord(mutation)

			So(record.ID, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestDecodeDMTDocument(t *testing.T) {
	Convey("Given encoded document record", t, func() {
		record := dmtDocumentRecord{
			ID:    "record-3",
			Scope: "goal-record",
			Text:  "Decode me.",
			Metadata: types.Metadata{
				ID:     "record-3",
				Source: "goal-record",
			},
		}

		payload, err := json.Marshal(record)
		So(err, ShouldBeNil)

		Convey("It should decode the record", func() {
			decoded, err := decodeDMTDocument(payload)

			So(err, ShouldBeNil)
			So(decoded.document().ID, ShouldEqual, "record-3")
			So(decoded.document().Metadata.Source, ShouldEqual, "goal-record")
		})
	})
}

func TestNewDMTRelationshipRecord(t *testing.T) {
	Convey("Given a relationship mutation", t, func() {
		mutation := types.Mutation{
			ID:           "from-1",
			RelatedID:    "to-1",
			Relationship: "supports",
			Metadata: types.Metadata{
				ID:     "rel-1",
				Source: "goal-record",
			},
		}

		Convey("It should build a relationship record", func() {
			record, err := newDMTRelationshipRecord(mutation)

			So(err, ShouldBeNil)
			So(record.relationshipKey(), ShouldContainSubstring, "goal_record")
			So(record.relationship().ID, ShouldEqual, "from-1")
			So(record.relationship().ToID, ShouldEqual, "to-1")
		})
	})
}

func TestMemorySuffixSequences(t *testing.T) {
	Convey("Given text with punctuation", t, func() {
		sequences := memorySuffixSequences("Make test: proof memory")

		Convey("It should produce underscore suffixes", func() {
			So(sequences, ShouldHaveLength, 4)
			So(string(sequences[0]), ShouldEqual, "make_test_proof_memory")
			So(string(sequences[2]), ShouldEqual, "proof_memory")
		})
	})
}

func TestDocumentKeyFromSequenceIndex(t *testing.T) {
	Convey("Given a sequence index key", t, func() {
		documentKey := documentKey("goal-key", "record-key")
		indexKey := sequenceIndexKey([]byte("proof_memory"), documentKey)

		Convey("It should extract the document key suffix", func() {
			So(documentKeyFromSequenceIndex([]byte(indexKey)), ShouldEqual, documentKey)
		})
	})
}
