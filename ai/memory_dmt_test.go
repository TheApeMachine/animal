package ai

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura/types"
	"github.com/theapemachine/errnie"
)

func TestNewDMTMemoryStore(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a context", t, func() {
		store, err := NewDMTMemoryStore(context.Background(), DMTMemoryConfig{})

		Convey("It should create cognitive memory", func() {
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.Close(), ShouldBeNil)
		})
	})

	Convey("Given a nil context", t, func() {
		var ctx context.Context

		Convey("It should reject construction", func() {
			store, err := NewDMTMemoryStore(ctx, DMTMemoryConfig{})

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestDMTMemoryStorePut(t *testing.T) {
	Convey("Given cognitive memory", t, func() {
		ctx := context.Background()
		store := mustDMTMemoryStore(t)
		defer store.Close()

		mutation := dmtMemoryMutation("dmt-put-1", "goal-dmt-put", "make test proof memory")

		Convey("It should store documents and train sensory recall", func() {
			err := store.Put(ctx, mutation)
			So(err, ShouldBeNil)

			tree, err := store.tree()
			So(err, ShouldBeNil)

			state := tree.GetSensoryWeight([]byte("proof_memory"))
			So(state.Count, ShouldBeGreaterThan, 0)
		})
	})
}

func TestDMTMemoryStoreGet(t *testing.T) {
	Convey("Given a stored cognitive memory document", t, func() {
		ctx := context.Background()
		store := mustDMTMemoryStore(t)
		defer store.Close()

		err := store.Put(ctx, dmtMemoryMutation("dmt-get-1", "goal-dmt-get", "exact cognitive recall"))
		So(err, ShouldBeNil)

		Convey("It should retrieve by ID", func() {
			recalled, err := store.Get(ctx, types.Query{ID: "dmt-get-1"})

			So(err, ShouldBeNil)
			So(recalled.Documents, ShouldHaveLength, 1)
			So(recalled.Documents[0].Text, ShouldEqual, "exact cognitive recall")
		})
	})
}

func TestDMTMemoryStoreSearch(t *testing.T) {
	Convey("Given cognitive memory across scopes", t, func() {
		ctx := context.Background()
		store := mustDMTMemoryStore(t)
		defer store.Close()

		So(store.Put(ctx, dmtMemoryMutation("dmt-search-1", "goal-dmt-search-a", "make test proof memory")), ShouldBeNil)
		So(store.Put(ctx, dmtMemoryMutation("dmt-search-2", "goal-dmt-search-b", "integration proof memory")), ShouldBeNil)

		Convey("It should recall through sensory beam search and scope filtering", func() {
			recalled, err := store.Search(ctx, types.Query{
				Text:  "proof",
				Limit: 4,
				Metadata: types.Metadata{
					Source: "goal-dmt-search-a",
				},
			})

			So(err, ShouldBeNil)
			So(recalled.Documents, ShouldHaveLength, 1)
			So(recalled.Documents[0].ID, ShouldEqual, "dmt-search-1")
		})
	})
}

func TestDMTMemoryStorePutRelationship(t *testing.T) {
	Convey("Given cognitive memory relationship", t, func() {
		ctx := context.Background()
		store := mustDMTMemoryStore(t)
		defer store.Close()

		mutation := types.Mutation{
			ID:           "dmt-rel-from",
			RelatedID:    "dmt-rel-to",
			Relationship: "supports",
			Metadata: types.Metadata{
				ID:     "dmt-rel-1",
				Source: "goal-dmt-rel",
			},
		}

		Convey("It should train an attractor basin and recall the relationship", func() {
			err := store.Put(ctx, mutation)
			So(err, ShouldBeNil)

			tree, err := store.tree()
			So(err, ShouldBeNil)

			state := tree.GetAttractorBasin([]byte("supports"), []byte("dmt_rel_from_dmt_rel_to"))
			So(state.Count, ShouldEqual, 1)

			recalled, err := store.Search(ctx, types.Query{
				Text:  "dmt rel from",
				Limit: 4,
				Metadata: types.Metadata{
					Source: "goal-dmt-rel",
				},
			})

			So(err, ShouldBeNil)
			So(recalled.Relationships, ShouldHaveLength, 1)
			So(recalled.Relationships[0].ID, ShouldEqual, "dmt-rel-from")
			So(recalled.Relationships[0].ToID, ShouldEqual, "dmt-rel-to")
		})
	})
}

func TestDMTMemoryStoreDelete(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a stored cognitive memory document", t, func() {
		ctx := context.Background()
		store := mustDMTMemoryStore(t)
		defer store.Close()

		So(store.Put(ctx, dmtMemoryMutation("dmt-delete-1", "goal-dmt-delete", "delete cognitive memory")), ShouldBeNil)

		Convey("It should tombstone the document", func() {
			err := store.Delete(ctx, types.Mutation{ID: "dmt-delete-1"})
			So(err, ShouldBeNil)

			recalled, err := store.Get(ctx, types.Query{ID: "dmt-delete-1"})

			So(recalled.Documents, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(errnie.IsNotFound(err), ShouldBeTrue)
		})
	})
}

func BenchmarkDMTMemoryStoreSearch(benchmark *testing.B) {
	ctx := context.Background()
	store := benchmarkDMTMemoryStore(benchmark)
	defer store.Close()

	for index := range 64 {
		mutation := dmtMemoryMutation(
			"dmt-bench-"+string(rune('a'+index)),
			"goal-dmt-bench",
			"make test proof memory",
		)

		if err := store.Put(ctx, mutation); err != nil {
			benchmark.Fatal(err)
		}
	}

	query := types.Query{
		Text:  "proof",
		Limit: 8,
		Metadata: types.Metadata{
			Source: "goal-dmt-bench",
		},
	}

	for benchmark.Loop() {
		if _, err := store.Search(ctx, query); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func mustDMTMemoryStore(t *testing.T) *DMTMemoryStore {
	t.Helper()

	store, err := NewDMTMemoryStore(context.Background(), DMTMemoryConfig{})
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func benchmarkDMTMemoryStore(benchmark *testing.B) *DMTMemoryStore {
	benchmark.Helper()

	store, err := NewDMTMemoryStore(context.Background(), DMTMemoryConfig{})
	if err != nil {
		benchmark.Fatal(err)
	}

	return store
}

func dmtMemoryMutation(id string, scope string, text string) types.Mutation {
	return types.Mutation{
		ID:   id,
		Text: text,
		Metadata: types.Metadata{
			ID:     id,
			Source: scope,
		},
	}
}
