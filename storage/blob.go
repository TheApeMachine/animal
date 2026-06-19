package storage

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/theapemachine/datura"
	daturas3 "github.com/theapemachine/datura/s3"
	"github.com/theapemachine/errnie"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
	"gocloud.dev/gcerrors"
)

/*
BlobConfig selects a URL-opened Go Cloud blob bucket.
Prefix is applied with blob.PrefixedBucket when non-empty.
*/
type BlobConfig struct {
	BucketURL string `yaml:"bucket_url" mapstructure:"bucket_url"`
	Prefix    string `yaml:"prefix" mapstructure:"prefix"`
}

/*
S3Config selects datura's S3-backed blob bucket.
*/
type S3Config struct {
	BucketURL string `yaml:"bucket_url" mapstructure:"bucket_url"`
	Bucket    string `yaml:"bucket" mapstructure:"bucket"`
	Region    string `yaml:"region" mapstructure:"region"`
	Prefix    string `yaml:"prefix" mapstructure:"prefix"`
}

/*
BlobStore stores datura artifacts in a Go Cloud blob bucket.
It keeps the same prefix contract as DMTStore for local, S3, and S3-compatible buckets.
*/
type BlobStore struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	bucket *blob.Bucket
	client *daturas3.Client
}

/*
NewBlobStore instantiates a blob-backed artifact store.
*/
func NewBlobStore(ctx context.Context, bucket *blob.Bucket) (*BlobStore, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "blob store context is required", nil)
	}

	if bucket == nil {
		return nil, errnie.Err(errnie.Validation, "blob store bucket is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	store := &BlobStore{
		ctx:    ctx,
		cancel: cancel,
		bucket: bucket,
	}

	return store, errnie.Require(map[string]any{
		"ctx":    store.ctx,
		"cancel": store.cancel,
		"bucket": store.bucket,
	})
}

/*
NewBlobStoreURL instantiates a blob-backed artifact store from a Go Cloud bucket URL.
*/
func NewBlobStoreURL(ctx context.Context, config BlobConfig) (*BlobStore, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "blob store context is required", nil)
	}

	bucketURL := strings.TrimSpace(config.BucketURL)
	if bucketURL == "" {
		return nil, errnie.Err(errnie.Validation, "blob store bucket URL is required", nil)
	}

	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, errnie.Err(errnie.IO, "blob store bucket open failed", err)
	}

	prefix := normalizePrefix(strings.TrimSpace(config.Prefix))
	if prefix != "" {
		bucket = blob.PrefixedBucket(bucket, prefix)
	}

	return NewBlobStore(ctx, bucket)
}

/*
NewS3Store instantiates an S3-backed artifact store using datura's S3 client.
*/
func NewS3Store(ctx context.Context, config S3Config) (*BlobStore, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "s3 store context is required", nil)
	}

	client, err := daturas3.NewClient(ctx, config.datura())
	if err != nil {
		return nil, errnie.Err(errnie.IO, "s3 store client create failed", err)
	}

	store, err := NewBlobStore(ctx, client.Bucket())
	if err != nil {
		if err := client.Close(); err != nil {
			errnie.Error(errnie.Err(errnie.IO, "s3 store client close failed", err))
		}

		return nil, err
	}

	store.client = client

	return store, nil
}

/*
Put stores an artifact under its datura prefix key and returns that key.
*/
func (store *BlobStore) Put(
	ctx context.Context,
	artifact *datura.Artifact,
) (string, error) {
	if artifact == nil {
		return "", errnie.Err(errnie.Validation, "blob store artifact is required", nil)
	}

	key := strings.TrimSpace(string(artifact.Prefix()))

	if key == "" || key == "." {
		return "", errnie.Err(errnie.Validation, "blob store artifact prefix is required", nil)
	}

	return key, store.PutKey(ctx, key, artifact)
}

/*
PutKey stores an artifact under an explicit prefix key.
*/
func (store *BlobStore) PutKey(
	ctx context.Context,
	key string,
	artifact *datura.Artifact,
) error {
	if err := validateRequest(ctx, key, "blob store"); err != nil {
		return err
	}

	if artifact == nil {
		return errnie.Err(errnie.Validation, "blob store artifact is required", nil)
	}

	payload := artifact.Pack()

	if len(payload) == 0 {
		return errnie.Err(errnie.Validation, "blob store artifact pack failed", nil)
	}

	if err := store.bucket.WriteAll(ctx, key, payload, nil); err != nil {
		return errnie.Err(errnie.IO, "blob store artifact write failed", err)
	}

	return nil
}

/*
Get retrieves one artifact by exact key.
*/
func (store *BlobStore) Get(
	ctx context.Context,
	key string,
) (*datura.Artifact, error) {
	if err := validateRequest(ctx, key, "blob store"); err != nil {
		return nil, err
	}

	payload, err := store.bucket.ReadAll(ctx, key)
	if err != nil {
		if gcerrors.Code(err) == gcerrors.NotFound {
			return nil, errnie.Err(errnie.NotFound, "blob store artifact not found", err)
		}

		return nil, errnie.Err(errnie.IO, "blob store artifact read failed", err)
	}

	return decodeArtifact(payload, "blob store")
}

/*
List returns artifacts whose keys share the requested prefix.
*/
func (store *BlobStore) List(
	ctx context.Context,
	prefix string,
) ([]Record, error) {
	if err := validateRequest(ctx, prefix, "blob store"); err != nil {
		return nil, err
	}

	iterator := store.bucket.List(&blob.ListOptions{Prefix: prefix})
	records := make([]Record, 0)

	for {
		object, err := iterator.Next(ctx)

		if errors.Is(err, io.EOF) {
			return records, nil
		}

		if err != nil {
			return nil, errnie.Err(errnie.IO, "blob store artifact list failed", err)
		}

		if object.IsDir {
			continue
		}

		artifact, err := store.Get(ctx, object.Key)
		if err != nil {
			return nil, err
		}

		records = append(records, Record{
			Key:      object.Key,
			Artifact: artifact,
		})
	}
}

/*
Close closes the blob bucket and cancels the store scope.
*/
func (store *BlobStore) Close() error {
	store.cancel()

	if store.client != nil {
		return store.client.Close()
	}

	return store.bucket.Close()
}

func (config S3Config) datura() daturas3.Config {
	return daturas3.Config{
		BucketURL: config.BucketURL,
		Bucket:    config.Bucket,
		Region:    config.Region,
		Prefix:    config.Prefix,
	}
}

func normalizePrefix(prefix string) string {
	if prefix == "" {
		return ""
	}

	if strings.HasSuffix(prefix, "/") {
		return prefix
	}

	return prefix + "/"
}
