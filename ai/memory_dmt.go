package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/theapemachine/datura/dmt"
	"github.com/theapemachine/datura/types"
	"github.com/theapemachine/errnie"
)

const (
	dmtMemoryDocumentPrefix     = "animal/memory/document/"
	dmtMemoryDocumentIndex      = "animal/memory/document-index/"
	dmtMemorySequenceIndex      = "animal/memory/sequence-index/"
	dmtMemoryRelationshipPrefix = "animal/memory/relationship/"
	dmtMemoryRelationshipIndex  = "animal/memory/relationship-index/"
	dmtMemoryTombstonePrefix    = "animal/memory/tombstone/"
	dmtMemorySensoryPrefix      = "s/"
)

/*
DMTMemoryConfig controls DMT-backed cognitive memory.
*/
type DMTMemoryConfig struct {
	PersistDir string
}

/*
DMTMemoryStore adapts DMT radix cognition to datura's memory Store interface.
*/
type DMTMemoryStore struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	forest *dmt.Forest
}

/*
NewDMTMemoryStore instantiates DMT-backed cognitive memory.
*/
func NewDMTMemoryStore(ctx context.Context, config DMTMemoryConfig) (*DMTMemoryStore, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "dmt memory context is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	forest, err := dmt.NewForest(dmt.ForestConfig{PersistDir: config.PersistDir})
	if err != nil {
		cancel()

		return nil, errnie.Err(errnie.IO, "dmt memory forest creation failed", err)
	}

	store := &DMTMemoryStore{
		ctx:    ctx,
		cancel: cancel,
		forest: forest,
	}

	return store, errnie.Require(map[string]any{
		"ctx":    store.ctx,
		"cancel": store.cancel,
		"forest": store.forest,
	})
}

/*
Get retrieves one memory document by ID.
*/
func (store *DMTMemoryStore) Get(
	ctx context.Context,
	query types.Query,
) (types.Memory, error) {
	if err := store.validate(ctx); err != nil {
		return types.Memory{}, err
	}

	documentKey, err := store.documentKey(query)
	if err != nil {
		return types.Memory{}, err
	}

	return store.getDocument(documentKey)
}

/*
Put writes a document or relationship into DMT memory.
*/
func (store *DMTMemoryStore) Put(
	ctx context.Context,
	mutation types.Mutation,
) error {
	if err := store.validate(ctx); err != nil {
		return err
	}

	if strings.TrimSpace(mutation.Relationship) != "" ||
		strings.TrimSpace(mutation.RelatedID) != "" {
		return store.putRelationship(mutation)
	}

	return store.putDocument(mutation)
}

/*
Delete tombstones a memory document by ID.
*/
func (store *DMTMemoryStore) Delete(
	ctx context.Context,
	mutation types.Mutation,
) error {
	if err := store.validate(ctx); err != nil {
		return err
	}

	if strings.TrimSpace(mutation.ID) == "" {
		return errnie.Err(errnie.Validation, "dmt memory delete ID is required", nil)
	}

	_, err := store.documentKey(types.Query{ID: mutation.ID})
	if err != nil {
		return err
	}

	store.forest.Insert([]byte(tombstoneKey(mutation.ID)), []byte{1})

	return nil
}

/*
Search recalls documents and relationships through DMT cognitive paths.
*/
func (store *DMTMemoryStore) Search(
	ctx context.Context,
	query types.Query,
) (types.Memory, error) {
	if err := store.validate(ctx); err != nil {
		return types.Memory{}, err
	}

	search := newDMTMemorySearch(store, query)
	if err := search.validate(); err != nil {
		return types.Memory{}, err
	}

	return search.run()
}

/*
Close closes the DMT forest and cancels the memory scope.
*/
func (store *DMTMemoryStore) Close() error {
	store.cancel()

	return store.forest.Close()
}

func (store *DMTMemoryStore) putDocument(mutation types.Mutation) error {
	record, err := newDMTDocumentRecord(mutation)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return errnie.Err(errnie.Validation, "dmt memory document marshal failed", err)
	}

	documentKey := record.documentKey()
	store.forest.Insert([]byte(documentKey), payload)
	store.forest.Insert([]byte(documentIndexKey(record.ID)), []byte(documentKey))

	if err := store.trainDocument(record, documentKey); err != nil {
		return err
	}

	return nil
}

func (store *DMTMemoryStore) putRelationship(mutation types.Mutation) error {
	record, err := newDMTRelationshipRecord(mutation)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(record)
	if err != nil {
		return errnie.Err(errnie.Validation, "dmt memory relationship marshal failed", err)
	}

	tree, err := store.tree()
	if err != nil {
		return err
	}

	relationshipKey := record.relationshipKey()
	sequence := memorySequence(record.FromID + " " + record.ToID)
	timestamp := uint64(time.Now().UnixNano())

	store.forest.Insert([]byte(relationshipKey), payload)
	for _, indexedSequence := range memorySuffixSequences(record.FromID + " " + record.ToID) {
		_, _ = tree.CommitToEpisodicBuffer(timestamp, indexedSequence)
		store.forest.Insert([]byte(relationshipIndexKey(indexedSequence, relationshipKey)), []byte{1})
	}

	tree.ExecuteREMSleepConsolidation(timestamp, timestamp)
	_, _ = tree.InsertAttractorBasin(
		[]byte(memoryToken(record.Relationship)),
		sequence,
		dmt.CognitiveState{Count: 1, Probability: record.Importance},
	)

	return nil
}

func (store *DMTMemoryStore) trainDocument(
	record dmtDocumentRecord,
	documentKey string,
) error {
	tree, err := store.tree()
	if err != nil {
		return err
	}

	timestamp := uint64(time.Now().UnixNano())

	for _, sequence := range memorySuffixSequences(record.Text) {
		_, _ = tree.CommitToEpisodicBuffer(timestamp, sequence)
		store.forest.Insert([]byte(sequenceIndexKey(sequence, documentKey)), []byte{1})
	}

	tree.ExecuteREMSleepConsolidation(timestamp, timestamp)

	return nil
}

func (store *DMTMemoryStore) getDocument(documentKey string) (types.Memory, error) {
	payload, found := store.forest.Get([]byte(documentKey))
	if !found {
		return types.Memory{}, errnie.Err(errnie.NotFound, "dmt memory document not found", nil)
	}

	record, err := decodeDMTDocument(payload)
	if err != nil {
		return types.Memory{}, err
	}

	if store.deleted(record.ID) {
		return types.Memory{}, errnie.Err(errnie.NotFound, "dmt memory document not found", nil)
	}

	out := types.NewMemory()
	out.AddDocument(record.document())

	return out, nil
}

func (store *DMTMemoryStore) documentKey(query types.Query) (string, error) {
	id := strings.TrimSpace(query.ID)
	if id == "" {
		return "", errnie.Err(errnie.Validation, "dmt memory query ID is required", nil)
	}

	scope := strings.TrimSpace(query.Metadata.Source)
	if scope != "" {
		return documentKey(scope, id), nil
	}

	documentKeyPayload, found := store.forest.Get([]byte(documentIndexKey(id)))
	if !found {
		return "", errnie.Err(errnie.NotFound, "dmt memory document not found", nil)
	}

	return string(documentKeyPayload), nil
}

func (store *DMTMemoryStore) tree() (*dmt.Tree, error) {
	tree := store.forest.GetFastestTree()
	if tree == nil {
		return nil, errnie.Err(errnie.Validation, "dmt memory tree is required", nil)
	}

	return tree, nil
}

func (store *DMTMemoryStore) validate(ctx context.Context) error {
	if ctx == nil {
		return errnie.Err(errnie.Validation, "dmt memory context is required", nil)
	}

	if store.ctx.Err() != nil {
		return errnie.Err(errnie.Timeout, "dmt memory store is closed", store.ctx.Err())
	}

	return nil
}

func (store *DMTMemoryStore) deleted(id string) bool {
	_, found := store.forest.Get([]byte(tombstoneKey(id)))

	return found
}
