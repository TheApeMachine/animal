package config

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/storage"
)

/*
TestStorageConfigFromViper verifies storage config extraction from viper.
*/
func TestStorageConfigFromViper(t *testing.T) {
	Convey("Given ai.storage values in viper", t, func() {
		viper.Reset()
		defer viper.Reset()

		viper.Set("ai.storage.driver", "blob")
		viper.Set("ai.storage.blob.bucket_url", "mem://")
		viper.Set("ai.storage.blob.prefix", "training")
		viper.Set("ai.storage.dmt.persist_dir", "/tmp/animal-dmt")
		viper.Set("ai.storage.s3.bucket", "training-bucket")
		viper.Set("ai.storage.s3.region", "us-east-1")

		Convey("When StorageConfigFromViper is called", func() {
			config := StorageConfigFromViper()

			Convey("Then it should return storage settings", func() {
				So(config.Driver, ShouldEqual, storage.DriverBlob)
				So(config.Blob.BucketURL, ShouldEqual, "mem://")
				So(config.Blob.Prefix, ShouldEqual, "training")
				So(config.DMT.PersistDir, ShouldEqual, "/tmp/animal-dmt")
				So(config.S3.Bucket, ShouldEqual, "training-bucket")
				So(config.S3.Region, ShouldEqual, "us-east-1")
			})
		})
	})
}

/*
TestArtifactStoreFromViper verifies opening the configured artifact store.
*/
func TestArtifactStoreFromViper(t *testing.T) {
	Convey("Given a blob storage config in viper", t, func() {
		viper.Reset()
		defer viper.Reset()

		viper.Set("ai.storage.driver", "blob")
		viper.Set("ai.storage.blob.bucket_url", "mem://")

		Convey("When ArtifactStoreFromViper is called", func() {
			store, err := ArtifactStoreFromViper(context.Background())

			Convey("Then it should open the configured store", func() {
				So(err, ShouldBeNil)
				So(store, ShouldHaveSameTypeAs, &storage.BlobStore{})
				So(store.Close(), ShouldBeNil)
			})
		})
	})
}

func BenchmarkStorageConfigFromViper(benchmark *testing.B) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("ai.storage.driver", "blob")
	viper.Set("ai.storage.blob.bucket_url", "mem://")
	viper.Set("ai.storage.blob.prefix", "training")

	for benchmark.Loop() {
		_ = StorageConfigFromViper()
	}
}

func BenchmarkArtifactStoreFromViper(benchmark *testing.B) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("ai.storage.driver", "blob")
	viper.Set("ai.storage.blob.bucket_url", "mem://")

	for benchmark.Loop() {
		store, err := ArtifactStoreFromViper(context.Background())

		if err != nil {
			benchmark.Fatal(err)
		}

		_ = store.Close()
	}
}
