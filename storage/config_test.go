package storage

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/errnie"
)

/*
TestNewStore verifies artifact store construction from config.
*/
func TestNewStore(t *testing.T) {
	restoreLogging := errnie.SuppressLogging()
	defer restoreLogging()

	Convey("Given a DMT storage config", t, func() {
		config := Config{Driver: DriverDMT}

		Convey("When NewStore is called", func() {
			store, err := NewStore(context.Background(), config)

			Convey("Then it should create a DMT store", func() {
				So(err, ShouldBeNil)
				So(store, ShouldHaveSameTypeAs, &DMTStore{})
				So(store.Close(), ShouldBeNil)
			})
		})
	})

	Convey("Given a blob URL storage config", t, func() {
		config := Config{
			Driver: DriverBlob,
			Blob:   BlobConfig{BucketURL: "mem://"},
		}

		Convey("When NewStore is called", func() {
			store, err := NewStore(context.Background(), config)

			Convey("Then it should create a blob store", func() {
				So(err, ShouldBeNil)
				So(store, ShouldHaveSameTypeAs, &BlobStore{})
				So(store.Close(), ShouldBeNil)
			})
		})
	})

	Convey("Given an invalid storage driver", t, func() {
		config := Config{Driver: Driver("missing")}

		Convey("When NewStore is called", func() {
			store, err := NewStore(context.Background(), config)

			Convey("Then it should reject the config", func() {
				So(store, ShouldBeNil)
				So(err, ShouldNotBeNil)
				So(errnie.IsValidation(err), ShouldBeTrue)
			})
		})
	})
}

/*
TestConfigStore verifies direct store construction from a Config value.
*/
func TestConfigStore(t *testing.T) {
	Convey("Given a normalized blob config", t, func() {
		config := Config{
			Driver: Driver(" BLOB "),
			Blob:   BlobConfig{BucketURL: "mem://"},
		}

		Convey("When Store is called", func() {
			store, err := config.Store(context.Background())

			Convey("Then it should create the selected store", func() {
				So(err, ShouldBeNil)
				So(store, ShouldHaveSameTypeAs, &BlobStore{})
				So(store.Close(), ShouldBeNil)
			})
		})
	})
}

func BenchmarkNewStoreDMT(benchmark *testing.B) {
	config := Config{Driver: DriverDMT}

	for benchmark.Loop() {
		store, err := NewStore(context.Background(), config)

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}

func BenchmarkConfigStore(benchmark *testing.B) {
	config := Config{
		Driver: DriverBlob,
		Blob:   BlobConfig{BucketURL: "mem://"},
	}

	for benchmark.Loop() {
		store, err := config.Store(context.Background())

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}

func BenchmarkNewStoreBlob(benchmark *testing.B) {
	config := Config{
		Driver: DriverBlob,
		Blob:   BlobConfig{BucketURL: "mem://"},
	}

	for benchmark.Loop() {
		store, err := NewStore(context.Background(), config)

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}
