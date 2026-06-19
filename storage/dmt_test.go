package storage

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/errnie"
)

func TestNewDMTStore(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a DMT store config", t, func() {
		store, err := NewDMTStore(context.Background(), DMTConfig{})

		Convey("It should create a store", func() {
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.tree, ShouldNotBeNil)
			So(store.Close(), ShouldBeNil)
		})
	})

	Convey("Given a DMT persistence directory", t, func() {
		store, err := NewDMTStore(
			context.Background(),
			DMTConfig{PersistDir: t.TempDir()},
		)

		Convey("It should create a persistent store", func() {
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.Close(), ShouldBeNil)
		})
	})

	Convey("Given a nil context", t, func() {
		var ctx context.Context

		Convey("It should reject construction", func() {
			store, err := NewDMTStore(ctx, DMTConfig{})

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestDMTStorePut(t *testing.T) {
	Convey("Given a DMT store and artifact", t, func() {
		store := mustDMTStore(t)
		defer store.Close()
		artifact := newStorageArtifact("agent-a", "developer", "goal-1", "hello")

		Convey("It should store the artifact under its datura prefix", func() {
			key, err := store.Put(context.Background(), artifact)

			So(err, ShouldBeNil)
			So(key, ShouldContainSubstring, "agent-a")
			So(key, ShouldContainSubstring, "developer")
			So(key, ShouldContainSubstring, "goal-1")

			stored, err := store.Get(context.Background(), key)
			So(err, ShouldBeNil)
			So(payloadString(stored), ShouldEqual, "hello")
		})
	})

	Convey("Given a nil artifact", t, func() {
		store := mustDMTStore(t)
		defer store.Close()

		Convey("It should reject storage", func() {
			key, err := store.Put(context.Background(), nil)

			So(key, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestDMTStorePutKey(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given an explicit key", t, func() {
		store := mustDMTStore(t)
		defer store.Close()
		artifact := newStorageArtifact("agent-a", "developer", "goal-2", "explicit")

		Convey("It should store the artifact under that key", func() {
			err := store.PutKey(context.Background(), "goals/goal-2/result.json", artifact)

			So(err, ShouldBeNil)

			stored, err := store.Get(context.Background(), "goals/goal-2/result.json")
			So(err, ShouldBeNil)
			So(payloadString(stored), ShouldEqual, "explicit")
		})
	})

	Convey("Given an invalid request", t, func() {
		store := mustDMTStore(t)
		defer store.Close()
		artifact := newStorageArtifact("agent-a", "developer", "goal-2", "explicit")
		var ctx context.Context

		Convey("It should reject the request", func() {
			err := store.PutKey(ctx, "goals/goal-2/result.json", artifact)
			So(err, ShouldNotBeNil)

			err = store.PutKey(context.Background(), "", artifact)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)

			err = store.PutKey(context.Background(), "goals/goal-2/result.json", nil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestDMTStoreGet(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a missing key", t, func() {
		store := mustDMTStore(t)
		defer store.Close()

		Convey("It should return not found", func() {
			artifact, err := store.Get(context.Background(), "missing")

			So(artifact, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsNotFound(err), ShouldBeTrue)
		})
	})
}

func TestDMTStoreList(t *testing.T) {
	Convey("Given artifacts under related prefixes", t, func() {
		store := mustDMTStore(t)
		defer store.Close()

		first := newStorageArtifact("agent-a", "developer", "goal-3", "first")
		second := newStorageArtifact("agent-b", "developer", "goal-3", "second")
		third := newStorageArtifact("agent-c", "reviewer", "goal-4", "third")

		So(store.PutKey(context.Background(), "goals/goal-3/first.json", first), ShouldBeNil)
		So(store.PutKey(context.Background(), "goals/goal-3/second.json", second), ShouldBeNil)
		So(store.PutKey(context.Background(), "goals/goal-4/third.json", third), ShouldBeNil)

		Convey("It should list only the matching prefix", func() {
			records, err := store.List(context.Background(), "goals/goal-3/")

			So(err, ShouldBeNil)
			So(records, ShouldHaveLength, 2)
			So(records[0].Key, ShouldEqual, "goals/goal-3/first.json")
			So(records[1].Key, ShouldEqual, "goals/goal-3/second.json")
			So(payloadString(records[0].Artifact), ShouldEqual, "first")
			So(payloadString(records[1].Artifact), ShouldEqual, "second")
		})
	})
}

func TestDMTStoreClose(t *testing.T) {
	Convey("Given a DMT store", t, func() {
		store := mustDMTStore(t)

		Convey("It should close the tree and cancel the context", func() {
			err := store.Close()

			So(err, ShouldBeNil)
			So(store.ctx.Err(), ShouldNotBeNil)
		})
	})
}

func TestDMTStorePersistence(t *testing.T) {
	Convey("Given a persisted DMT store", t, func() {
		persistDir := filepath.Join(t.TempDir(), "dmt")
		firstStore := mustDMTStoreWithConfig(t, DMTConfig{PersistDir: persistDir})
		artifact := newStorageArtifact("agent-a", "developer", "goal-5", "persisted")

		So(firstStore.PutKey(context.Background(), "goals/goal-5/result.json", artifact), ShouldBeNil)
		So(firstStore.Close(), ShouldBeNil)

		Convey("It should replay artifacts into a new store", func() {
			secondStore := mustDMTStoreWithConfig(t, DMTConfig{PersistDir: persistDir})
			defer secondStore.Close()

			stored, err := secondStore.Get(context.Background(), "goals/goal-5/result.json")

			So(err, ShouldBeNil)
			So(payloadString(stored), ShouldEqual, "persisted")
		})
	})
}

func BenchmarkNewDMTStore(benchmark *testing.B) {
	for benchmark.Loop() {
		store, err := NewDMTStore(context.Background(), DMTConfig{})

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}

func BenchmarkDMTStorePut(benchmark *testing.B) {
	store := benchmarkDMTStore(benchmark)
	defer store.Close()

	for benchmark.Loop() {
		artifact := newStorageArtifact("bench-agent", "developer", "goal", "payload")

		if _, err := store.Put(context.Background(), artifact); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkDMTStorePutKey(benchmark *testing.B) {
	store := benchmarkDMTStore(benchmark)
	defer store.Close()
	artifact := newStorageArtifact("bench-agent", "developer", "goal", "payload")

	for benchmark.Loop() {
		if err := store.PutKey(context.Background(), "bench/key.json", artifact); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkDMTStoreGet(benchmark *testing.B) {
	store := benchmarkDMTStore(benchmark)
	defer store.Close()
	artifact := newStorageArtifact("bench-agent", "developer", "goal", "payload")

	if err := store.PutKey(context.Background(), "bench/get.json", artifact); err != nil {
		benchmark.Fatal(err)
	}

	for benchmark.Loop() {
		if _, err := store.Get(context.Background(), "bench/get.json"); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkDMTStoreList(benchmark *testing.B) {
	store := benchmarkDMTStore(benchmark)
	defer store.Close()

	for index := range 8 {
		artifact := newStorageArtifact("bench-agent", "developer", "goal", "payload")
		key := "bench/list/" + string(rune('a'+index)) + ".json"

		if err := store.PutKey(context.Background(), key, artifact); err != nil {
			benchmark.Fatal(err)
		}
	}

	for benchmark.Loop() {
		if _, err := store.List(context.Background(), "bench/list/"); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkDMTStoreClose(benchmark *testing.B) {
	for benchmark.Loop() {
		store := benchmarkDMTStore(benchmark)

		if err := store.Close(); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func mustDMTStore(t *testing.T) *DMTStore {
	return mustDMTStoreWithConfig(t, DMTConfig{})
}

func mustDMTStoreWithConfig(t *testing.T, config DMTConfig) *DMTStore {
	t.Helper()

	store, err := NewDMTStore(context.Background(), config)

	if err != nil {
		t.Fatal(err)
	}

	return store
}

func benchmarkDMTStore(benchmark *testing.B) *DMTStore {
	benchmark.Helper()

	store, err := NewDMTStore(context.Background(), DMTConfig{})

	if err != nil {
		benchmark.Fatal(err)
	}

	return store
}

func newStorageArtifact(
	origin string,
	role string,
	scope string,
	payload string,
) *datura.Artifact {
	return datura.Acquire(origin, datura.Artifact_Type_json).
		WithRole(role).
		WithScope(scope).
		WithPayload([]byte(payload))
}

func payloadString(artifact *datura.Artifact) string {
	payload, err := artifact.DecryptPayloadError()

	if err != nil {
		return ""
	}

	return string(payload)
}
