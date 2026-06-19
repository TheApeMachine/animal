package ai

import (
	"bytes"
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/theapemachine/animal/storage"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/errnie"
)

const (
	trainingArtifactRole      = "training"
	trainingArtifactExtension = "jsonl"
	trainingArtifactTypeJSONL = datura.Artifact_Type_jsonl
)

/*
TrainingStore records successful agent traces as datura JSONL artifacts.
*/
type TrainingStore struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	store  storage.ArtifactStore
}

/*
NewTrainingStore instantiates an artifact-backed training recorder.
*/
func NewTrainingStore(ctx context.Context, store storage.ArtifactStore) (*TrainingStore, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "training store context is required", nil)
	}

	if store == nil {
		return nil, errnie.Err(errnie.Validation, "training artifact store is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	trainingStore := &TrainingStore{
		ctx:    ctx,
		cancel: cancel,
		store:  store,
	}

	return trainingStore, errnie.Require(map[string]any{
		"ctx":    trainingStore.ctx,
		"cancel": trainingStore.cancel,
		"store":  trainingStore.store,
	})
}

/*
NewTrainingStoreFromConfig instantiates an artifact-backed training recorder from storage config.
*/
func NewTrainingStoreFromConfig(
	ctx context.Context,
	config storage.Config,
) (*TrainingStore, error) {
	artifactStore, err := storage.NewStore(ctx, config)
	if err != nil {
		return nil, err
	}

	return NewTrainingStore(ctx, artifactStore)
}

/*
Record stores the agent trace when metric marks the outcome successful.
*/
func (trainingStore *TrainingStore) Record(
	agent *Agent,
	metric swarm.Metric,
) error {
	if strings.TrimSpace(metric.GoalID) == "" {
		return errnie.Err(errnie.Validation, "training metric goal ID is required", nil)
	}

	example, err := agent.FineTuneExample(metric)
	if err != nil {
		return err
	}

	_, err = trainingStore.RecordExample(metric.ActorID, metric.GoalID, example)

	return err
}

/*
RecordExample stores one validated JSONL record as a datura artifact.
*/
func (trainingStore *TrainingStore) RecordExample(
	actorID string,
	goalID string,
	example FineTuneExample,
) (string, error) {
	actorID = strings.TrimSpace(actorID)
	goalID = strings.TrimSpace(goalID)

	if actorID == "" {
		return "", errnie.Err(errnie.Validation, "training actor ID is required", nil)
	}

	if goalID == "" {
		return "", errnie.Err(errnie.Validation, "training goal ID is required", nil)
	}

	payload, err := example.JSONL()
	if err != nil {
		return "", err
	}

	artifact := datura.Acquire(actorID, trainingArtifactTypeJSONL)
	if artifact == nil {
		return "", errnie.Err(errnie.Validation, "training artifact allocation failed", nil)
	}

	artifact = artifact.WithRole(trainingArtifactRole).
		WithScope(goalID).
		WithPayload(payload)

	if artifact == nil {
		return "", errnie.Err(errnie.Validation, "training artifact payload failed", nil)
	}

	key, err := trainingStore.key(artifact)
	if err != nil {
		return "", err
	}

	return key, trainingStore.store.PutKey(trainingStore.ctx, key, artifact)
}

/*
ExportGoal concatenates all JSONL records stored for one goal.
*/
func (trainingStore *TrainingStore) ExportGoal(
	ctx context.Context,
	goalID string,
) ([]byte, error) {
	goalID = strings.TrimSpace(goalID)

	if goalID == "" {
		return nil, errnie.Err(errnie.Validation, "training goal ID is required", nil)
	}

	return trainingStore.Export(ctx, trainingArtifactRole+"/"+goalID+"/")
}

/*
Export concatenates JSONL records under a storage prefix.
*/
func (trainingStore *TrainingStore) Export(
	ctx context.Context,
	prefix string,
) ([]byte, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "training export context is required", nil)
	}

	if strings.TrimSpace(prefix) == "" {
		return nil, errnie.Err(errnie.Validation, "training export prefix is required", nil)
	}

	records, err := trainingStore.store.List(ctx, prefix)
	if err != nil {
		return nil, err
	}

	sort.Slice(records, func(left int, right int) bool {
		return records[left].Key < records[right].Key
	})

	var buffer bytes.Buffer

	for _, record := range records {
		if record.Artifact == nil {
			return nil, errnie.Err(errnie.Validation, "training artifact is required", nil)
		}

		payload, err := record.Artifact.DecryptPayloadError()
		if err != nil {
			return nil, errnie.Err(errnie.Validation, "training artifact payload read failed", err)
		}

		if len(payload) == 0 || payload[len(payload)-1] != '\n' {
			return nil, errnie.Err(errnie.Validation, "training artifact payload must be JSONL", nil)
		}

		if _, err := buffer.Write(payload); err != nil {
			return nil, errnie.Err(errnie.IO, "training export write failed", err)
		}
	}

	return buffer.Bytes(), nil
}

/*
Close closes the composed artifact store and cancels the training store scope.
*/
func (trainingStore *TrainingStore) Close() error {
	trainingStore.cancel()

	return trainingStore.store.Close()
}

func (trainingStore *TrainingStore) key(artifact *datura.Artifact) (string, error) {
	if artifact == nil {
		return "", errnie.Err(errnie.Validation, "training artifact is required", nil)
	}

	role, err := artifact.Role()
	if err != nil || strings.TrimSpace(role) == "" {
		return "", errnie.Err(errnie.Validation, "training artifact role is required", err)
	}

	scope, err := artifact.Scope()
	if err != nil || strings.TrimSpace(scope) == "" {
		return "", errnie.Err(errnie.Validation, "training artifact scope is required", err)
	}

	origin, err := artifact.Origin()
	if err != nil || strings.TrimSpace(origin) == "" {
		return "", errnie.Err(errnie.Validation, "training artifact origin is required", err)
	}

	uuidBytes, err := artifact.Uuid()
	uuid := string(uuidBytes)

	if err != nil || strings.TrimSpace(uuid) == "" {
		return "", errnie.Err(errnie.Validation, "training artifact UUID is required", err)
	}

	timestamp := artifact.Timestamp()

	if timestamp <= 0 {
		return "", errnie.Err(errnie.Validation, "training artifact timestamp is required", nil)
	}

	if artifact.Type() != trainingArtifactTypeJSONL {
		return "", errnie.Err(errnie.Validation, "training artifact type must be jsonl", nil)
	}

	return strings.Join([]string{
		role,
		scope,
		origin,
		strconv.FormatInt(timestamp, 36),
		uuid + "." + trainingArtifactExtension,
	}, "/"), nil
}
