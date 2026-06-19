package config

import (
	"context"

	"github.com/spf13/viper"
	"github.com/theapemachine/animal/storage"
)

/*
StorageConfigFromViper reads ai.storage from the active viper config.
*/
func StorageConfigFromViper() storage.Config {
	return storage.Config{
		Driver: storage.Driver(viper.GetString("ai.storage.driver")),
		DMT: storage.DMTConfig{
			PersistDir: viper.GetString("ai.storage.dmt.persist_dir"),
		},
		Blob: storage.BlobConfig{
			BucketURL: viper.GetString("ai.storage.blob.bucket_url"),
			Prefix:    viper.GetString("ai.storage.blob.prefix"),
		},
		S3: storage.S3Config{
			BucketURL: viper.GetString("ai.storage.s3.bucket_url"),
			Bucket:    viper.GetString("ai.storage.s3.bucket"),
			Region:    viper.GetString("ai.storage.s3.region"),
			Prefix:    viper.GetString("ai.storage.s3.prefix"),
		},
	}
}

/*
ArtifactStoreFromViper opens the configured artifact store from ai.storage.
*/
func ArtifactStoreFromViper(ctx context.Context) (storage.ArtifactStore, error) {
	return storage.NewStore(ctx, StorageConfigFromViper())
}
