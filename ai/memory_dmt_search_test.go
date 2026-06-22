package ai

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura/types"
	"github.com/theapemachine/errnie"
)

func TestNewDMTMemorySearch(t *testing.T) {
	Convey("Given a query without explicit limit", t, func() {
		store := mustDMTMemoryStore(t)
		defer store.Close()

		search := newDMTMemorySearch(store, types.Query{Text: "proof memory"})

		Convey("It should set default search state", func() {
			So(search.limit, ShouldEqual, 8)
			So(search.sequences, ShouldHaveLength, 2)
		})
	})
}

func TestDMTMemorySearchValidate(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given empty search text", t, func() {
		store := mustDMTMemoryStore(t)
		defer store.Close()

		search := newDMTMemorySearch(store, types.Query{Text: "..."})

		Convey("It should return validation error", func() {
			So(search.validate(), ShouldNotBeNil)
		})
	})
}

func TestDMTMemorySearchCandidates(t *testing.T) {
	Convey("Given trained cognitive memory", t, func() {
		ctx := context.Background()
		store := mustDMTMemoryStore(t)
		defer store.Close()

		So(store.Put(ctx, dmtMemoryMutation("candidate-1", "goal-candidate", "blue cab big")), ShouldBeNil)

		search := newDMTMemorySearch(store, types.Query{Text: "blue", Limit: 4})
		So(search.validate(), ShouldBeNil)

		Convey("It should include beam-search continuations", func() {
			candidates := search.candidates()

			So(candidates, ShouldNotBeEmpty)
			So(string(candidates[0]), ShouldEqual, "blue")
		})
	})
}

func TestDMTMemorySearchMatchesScope(t *testing.T) {
	Convey("Given a scoped query", t, func() {
		store := mustDMTMemoryStore(t)
		defer store.Close()

		search := newDMTMemorySearch(store, types.Query{
			Text: "proof",
			Metadata: types.Metadata{
				Source: "goal-a",
			},
		})

		Convey("It should match only the requested scope", func() {
			So(search.matchesScope("goal-a"), ShouldBeTrue)
			So(search.matchesScope("goal-b"), ShouldBeFalse)
		})
	})
}
