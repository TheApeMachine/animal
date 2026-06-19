package storage

import (
	"context"
	"strings"

	"github.com/theapemachine/errnie"
)

/*
Driver identifies the artifact store backend selected by config.
*/
type Driver string

const (
	DriverDMT  Driver = "dmt"
	DriverBlob Driver = "blob"
	DriverS3   Driver = "s3"
)

/*
Config selects one artifact store backend and its backend-specific settings.
*/
type Config struct {
	Driver Driver     `yaml:"driver" mapstructure:"driver"`
	DMT    DMTConfig  `yaml:"dmt" mapstructure:"dmt"`
	Blob   BlobConfig `yaml:"blob" mapstructure:"blob"`
	S3     S3Config   `yaml:"s3" mapstructure:"s3"`
}

/*
NewStore instantiates an artifact store from config.
*/
func NewStore(ctx context.Context, config Config) (ArtifactStore, error) {
	return config.Store(ctx)
}

/*
Store instantiates the configured artifact store.
*/
func (config Config) Store(ctx context.Context) (ArtifactStore, error) {
	driver := Driver(strings.ToLower(strings.TrimSpace(string(config.Driver))))

	switch driver {
	case DriverDMT:
		return NewDMTStore(ctx, config.DMT)
	case DriverBlob:
		return NewBlobStoreURL(ctx, config.Blob)
	case DriverS3:
		return NewS3Store(ctx, config.S3)
	default:
		return nil, errnie.Err(errnie.Validation, "storage driver is required", nil)
	}
}
