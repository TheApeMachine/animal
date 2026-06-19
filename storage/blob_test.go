package storage

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/errnie"
	"gocloud.dev/blob/memblob"
)

func TestNewBlobStore(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a blob bucket", t, func() {
		bucket := memblob.OpenBucket(nil)
		store, err := NewBlobStore(context.Background(), bucket)

		Convey("It should create a store", func() {
			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.bucket, ShouldNotBeNil)
			So(store.Close(), ShouldBeNil)
		})
	})

	Convey("Given a nil context", t, func() {
		var ctx context.Context

		Convey("It should reject construction", func() {
			store, err := NewBlobStore(ctx, memblob.OpenBucket(nil))

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})

	Convey("Given a nil bucket", t, func() {
		Convey("It should reject construction", func() {
			store, err := NewBlobStore(context.Background(), nil)

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestNewBlobStoreURL(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a blob bucket URL", t, func() {
		config := BlobConfig{BucketURL: "mem://", Prefix: "training"}

		Convey("It should create a blob store from the URL", func() {
			store, err := NewBlobStoreURL(context.Background(), config)

			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.bucket, ShouldNotBeNil)
			So(store.Close(), ShouldBeNil)
		})
	})

	Convey("Given a missing bucket URL", t, func() {
		Convey("It should reject construction", func() {
			store, err := NewBlobStoreURL(context.Background(), BlobConfig{})

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestNewS3Store(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a nil context", t, func() {
		var ctx context.Context

		Convey("It should reject construction", func() {
			store, err := NewS3Store(ctx, S3Config{})

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})

	Convey("Given an invalid S3 config", t, func() {
		Convey("It should return the datura S3 configuration error", func() {
			store, err := NewS3Store(context.Background(), S3Config{})

			So(store, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})
}

func TestBlobStorePut(t *testing.T) {
	Convey("Given a blob store and artifact", t, func() {
		store := mustBlobStore(t)
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
		store := mustBlobStore(t)
		defer store.Close()

		Convey("It should reject storage", func() {
			key, err := store.Put(context.Background(), nil)

			So(key, ShouldBeEmpty)
			So(err, ShouldNotBeNil)
			So(errnie.IsValidation(err), ShouldBeTrue)
		})
	})
}

func TestBlobStorePutKey(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given an explicit key", t, func() {
		store := mustBlobStore(t)
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
		store := mustBlobStore(t)
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

func TestBlobStoreGet(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a missing key", t, func() {
		store := mustBlobStore(t)
		defer store.Close()

		Convey("It should return not found", func() {
			artifact, err := store.Get(context.Background(), "missing")

			So(artifact, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(errnie.IsNotFound(err), ShouldBeTrue)
		})
	})
}

func TestBlobStoreList(t *testing.T) {
	Convey("Given artifacts under related prefixes", t, func() {
		store := mustBlobStore(t)
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

func TestBlobStoreClose(t *testing.T) {
	Convey("Given a blob store", t, func() {
		store := mustBlobStore(t)

		Convey("It should close the bucket and cancel the context", func() {
			err := store.Close()

			So(err, ShouldBeNil)
			So(store.ctx.Err(), ShouldNotBeNil)
		})
	})
}

func BenchmarkNewBlobStore(benchmark *testing.B) {
	for benchmark.Loop() {
		store, err := NewBlobStore(context.Background(), memblob.OpenBucket(nil))

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}

func BenchmarkNewBlobStoreURL(benchmark *testing.B) {
	config := BlobConfig{BucketURL: "mem://"}

	for benchmark.Loop() {
		store, err := NewBlobStoreURL(context.Background(), config)

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}

func BenchmarkNewS3StoreInvalidConfig(benchmark *testing.B) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	for benchmark.Loop() {
		store, err := NewS3Store(context.Background(), S3Config{})

		if err == nil {
			_ = store.Close()
			benchmark.Fatal("expected invalid S3 config")
		}
	}
}

func BenchmarkBlobStorePut(benchmark *testing.B) {
	store := benchmarkBlobStore(benchmark)
	defer store.Close()

	for benchmark.Loop() {
		artifact := newStorageArtifact("bench-agent", "developer", "goal", "payload")

		if _, err := store.Put(context.Background(), artifact); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkBlobStorePutKey(benchmark *testing.B) {
	store := benchmarkBlobStore(benchmark)
	defer store.Close()
	artifact := newStorageArtifact("bench-agent", "developer", "goal", "payload")

	for benchmark.Loop() {
		if err := store.PutKey(context.Background(), "bench/key.json", artifact); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func BenchmarkBlobStoreGet(benchmark *testing.B) {
	store := benchmarkBlobStore(benchmark)
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

func BenchmarkBlobStoreList(benchmark *testing.B) {
	store := benchmarkBlobStore(benchmark)
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

func BenchmarkBlobStoreClose(benchmark *testing.B) {
	for benchmark.Loop() {
		store := benchmarkBlobStore(benchmark)

		if err := store.Close(); err != nil {
			benchmark.Fatal(err)
		}
	}
}

func mustBlobStore(t *testing.T) *BlobStore {
	t.Helper()

	store, err := NewBlobStore(context.Background(), memblob.OpenBucket(nil))

	if err != nil {
		t.Fatal(err)
	}

	return store
}

func benchmarkBlobStore(benchmark *testing.B) *BlobStore {
	benchmark.Helper()

	store, err := NewBlobStore(context.Background(), memblob.OpenBucket(nil))

	if err != nil {
		benchmark.Fatal(err)
	}

	return store
}
