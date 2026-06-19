package storage

import (
	"context"
	"strings"

	"github.com/theapemachine/datura"
	"github.com/theapemachine/datura/dmt"
	"github.com/theapemachine/errnie"
)

/*
ArtifactStore is the storage contract for datura artifacts.
*/
type ArtifactStore interface {
	Put(ctx context.Context, artifact *datura.Artifact) (string, error)
	PutKey(ctx context.Context, key string, artifact *datura.Artifact) error
	Get(ctx context.Context, key string) (*datura.Artifact, error)
	List(ctx context.Context, prefix string) ([]Record, error)
	Close() error
}

/*
Record is one artifact returned from prefix storage.
It keeps the storage key beside the decoded datura artifact.
*/
type Record struct {
	Key      string
	Artifact *datura.Artifact
}

/*
DMTConfig controls the datura DMT-backed artifact store.
PersistDir enables DMT's write-ahead log when non-empty.
*/
type DMTConfig struct {
	PersistDir string `yaml:"persist_dir" mapstructure:"persist_dir"`
}

/*
DMTStore stores datura artifacts in datura's radix trie package.
It uses Artifact.Prefix as the default key, which is also suitable for S3-style prefix storage.
*/
type DMTStore struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	tree   *dmt.Tree
}

/*
NewDMTStore instantiates a DMT-backed artifact store.
*/
func NewDMTStore(ctx context.Context, config DMTConfig) (*DMTStore, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "dmt store context is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	tree := dmt.NewTree(config.PersistDir)

	store := &DMTStore{
		ctx:    ctx,
		cancel: cancel,
		tree:   tree,
	}

	return store, errnie.Require(map[string]any{
		"ctx":    store.ctx,
		"cancel": store.cancel,
		"tree":   store.tree,
	})
}

/*
Put stores an artifact under its datura prefix key and returns that key.
*/
func (store *DMTStore) Put(
	ctx context.Context,
	artifact *datura.Artifact,
) (string, error) {
	if artifact == nil {
		return "", errnie.Err(errnie.Validation, "dmt store artifact is required", nil)
	}

	key := strings.TrimSpace(string(artifact.Prefix()))

	if key == "" || key == "." {
		return "", errnie.Err(errnie.Validation, "dmt store artifact prefix is required", nil)
	}

	return key, store.PutKey(ctx, key, artifact)
}

/*
PutKey stores an artifact under an explicit prefix key.
*/
func (store *DMTStore) PutKey(
	ctx context.Context,
	key string,
	artifact *datura.Artifact,
) error {
	if err := validateRequest(ctx, key, "dmt store"); err != nil {
		return err
	}

	if artifact == nil {
		return errnie.Err(errnie.Validation, "dmt store artifact is required", nil)
	}

	payload := artifact.Pack()

	if len(payload) == 0 {
		return errnie.Err(errnie.Validation, "dmt store artifact pack failed", nil)
	}

	_, ok := store.tree.Insert([]byte(key), payload)

	if !ok {
		return errnie.Err(errnie.Conflict, "dmt store artifact was not inserted", nil)
	}

	return nil
}

/*
Get retrieves one artifact by exact key.
*/
func (store *DMTStore) Get(
	ctx context.Context,
	key string,
) (*datura.Artifact, error) {
	if err := validateRequest(ctx, key, "dmt store"); err != nil {
		return nil, err
	}

	payload, ok := store.tree.Get([]byte(key))

	if !ok {
		return nil, errnie.Err(errnie.NotFound, "dmt store artifact not found", nil)
	}

	return decodeArtifact(payload, "dmt store")
}

/*
List returns artifacts whose keys share the requested prefix.
*/
func (store *DMTStore) List(
	ctx context.Context,
	prefix string,
) ([]Record, error) {
	if err := validateRequest(ctx, prefix, "dmt store"); err != nil {
		return nil, err
	}

	records := make([]Record, 0)
	var (
		artifact *datura.Artifact
		err      error
	)

	store.tree.WalkPrefix([]byte(prefix), func(key []byte, value []byte) bool {
		artifact, err = decodeArtifact(value, "dmt store")

		if err != nil {
			return false
		}

		records = append(records, Record{
			Key:      string(key),
			Artifact: artifact,
		})

		return true
	})

	if err != nil {
		return nil, err
	}

	return records, nil
}

/*
Close closes the DMT tree and cancels the store scope.
*/
func (store *DMTStore) Close() error {
	store.cancel()

	return store.tree.Close()
}

func validateRequest(ctx context.Context, key string, storeName string) error {
	if ctx == nil {
		return errnie.Err(errnie.Validation, storeName+" context is required", nil)
	}

	if strings.TrimSpace(key) == "" {
		return errnie.Err(errnie.Validation, storeName+" key is required", nil)
	}

	return nil
}

func decodeArtifact(payload []byte, storeName string) (*datura.Artifact, error) {
	if len(payload) == 0 {
		return nil, errnie.Err(errnie.Validation, storeName+" payload is required", nil)
	}

	artifact := datura.Acquire(storeName, datura.Artifact_Type_json)

	if artifact == nil {
		return nil, errnie.Err(errnie.Validation, storeName+" artifact allocation failed", nil)
	}

	if err := artifact.Unpack(payload); err != nil {
		return nil, errnie.Err(errnie.Validation, storeName+" artifact unpack failed", err)
	}

	return artifact, nil
}
