package ai

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/errnie"
)

/*
TestNewLocalMemory verifies local datura memory construction.
*/
func TestNewLocalMemory(t *testing.T) {
	Convey("Given a context", t, func() {
		memory, err := NewLocalMemory(context.Background())

		Convey("It should create local memory", func() {
			So(err, ShouldBeNil)
			So(memory, ShouldNotBeNil)
			So(memory.Close(), ShouldBeNil)
		})
	})
}

/*
TestDaturaMemoryRecall verifies datura-backed memory recall.
*/
func TestDaturaMemoryRecall(t *testing.T) {
	Convey("Given remembered records in different scopes", t, func() {
		ctx := context.Background()
		memory, err := NewLocalMemory(ctx)
		So(err, ShouldBeNil)
		defer memory.Close()

		So(memory.Remember(ctx, MemoryConsolidation{
			Records: []MemoryRecord{
				{
					ID:         "goal-1:test-proof",
					Scope:      "goal-1",
					Text:       "Use make test output as proof.",
					Importance: 0.9,
				},
				{
					ID:         "goal-2:test-proof",
					Scope:      "goal-2",
					Text:       "Use integration output as proof.",
					Importance: 0.8,
				},
			},
		}), ShouldBeNil)

		Convey("When Recall searches one scope", func() {
			packet, err := memory.Recall(ctx, MemoryRecallPlan{
				Queries: []MemoryQuery{
					{
						Scope:      "goal-1",
						Text:       "proof",
						Limit:      4,
						TextWeight: 1,
					},
				},
			})

			Convey("Then it should return only matching scoped memory", func() {
				So(err, ShouldBeNil)
				So(packet.Documents, ShouldHaveLength, 1)
				So(packet.Documents[0].ID, ShouldEqual, "goal-1:test-proof")
			})
		})
	})
}

/*
TestDaturaMemoryRememberRejectsRelationshipWithoutGraph verifies graph writes are explicit.
*/
func TestDaturaMemoryRememberRejectsRelationshipWithoutGraph(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given datura memory without graph store", t, func() {
		ctx := context.Background()
		store, err := NewDMTMemoryStore(ctx, DMTMemoryConfig{})
		So(err, ShouldBeNil)
		defer store.Close()

		memory, err := NewDaturaMemory(ctx, store)
		So(err, ShouldBeNil)
		defer memory.Close()

		Convey("When Remember includes a relationship", func() {
			err := memory.Remember(ctx, MemoryConsolidation{
				Relationships: []MemoryRelationship{
					{
						ID:           "rel-1",
						Scope:        "goal-1",
						FromID:       "a",
						ToID:         "b",
						Relationship: "SUPPORTS",
						Importance:   0.7,
					},
				},
			})

			Convey("Then it should require a graph store", func() {
				So(err, ShouldNotBeNil)
				So(errnie.IsValidation(err), ShouldBeTrue)
			})
		})
	})
}

func BenchmarkDaturaMemoryRecall(benchmark *testing.B) {
	ctx := context.Background()
	memory, err := NewLocalMemory(ctx)
	if err != nil {
		benchmark.Fatal(err)
	}
	defer memory.Close()

	for index := range 64 {
		err := memory.Remember(ctx, MemoryConsolidation{
			Records: []MemoryRecord{
				{
					ID:         "goal-1:record-" + string(rune('a'+index)),
					Scope:      "goal-1",
					Text:       "make test proof memory",
					Importance: 0.7,
				},
			},
		})

		if err != nil {
			benchmark.Fatal(err)
		}
	}

	plan := MemoryRecallPlan{
		Queries: []MemoryQuery{
			{Scope: "goal-1", Text: "proof", Limit: 8, TextWeight: 1},
		},
	}

	for benchmark.Loop() {
		if _, err := memory.Recall(ctx, plan); err != nil {
			benchmark.Fatal(err)
		}
	}
}
