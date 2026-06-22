package ai

import (
	"context"

	"github.com/theapemachine/errnie"
)

/*
MemoryProjection is one latent memory signal from an external projector.
*/
type MemoryProjection struct {
	Embedding []float32
	Energy    float64
	Surprise  float64
}

/*
MemoryProjector projects text into latent memory signals.
*/
type MemoryProjector interface {
	Project(ctx context.Context, text string) (MemoryProjection, error)
	ProjectBatch(ctx context.Context, texts []string) ([]MemoryProjection, error)
	Close() error
}

/*
ProjectedMemory enriches consolidation records before delegating to memory.
*/
type ProjectedMemory struct {
	ctx       context.Context
	cancel    context.CancelFunc
	err       error
	memory    Memory
	projector MemoryProjector
}

/*
NewProjectedMemory instantiates projection-aware memory over an existing backend.
*/
func NewProjectedMemory(
	ctx context.Context,
	memory Memory,
	projector MemoryProjector,
) (*ProjectedMemory, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "projected memory context is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	projectedMemory := &ProjectedMemory{
		ctx:       ctx,
		cancel:    cancel,
		memory:    memory,
		projector: projector,
	}

	return projectedMemory, errnie.Require(map[string]any{
		"ctx":       projectedMemory.ctx,
		"cancel":    projectedMemory.cancel,
		"memory":    projectedMemory.memory,
		"projector": projectedMemory.projector,
	})
}

/*
Recall delegates recall to the wrapped memory.
*/
func (projectedMemory *ProjectedMemory) Recall(
	ctx context.Context,
	plan MemoryRecallPlan,
) (MemoryPacket, error) {
	return projectedMemory.memory.Recall(ctx, plan)
}

/*
Remember projects records and stores the enriched consolidation.
*/
func (projectedMemory *ProjectedMemory) Remember(
	ctx context.Context,
	consolidation MemoryConsolidation,
) error {
	enriched, err := projectedMemory.project(ctx, consolidation)
	if err != nil {
		return err
	}

	return projectedMemory.memory.Remember(ctx, enriched)
}

/*
Forget delegates forget to the wrapped memory.
*/
func (projectedMemory *ProjectedMemory) Forget(ctx context.Context, ids []string) error {
	return projectedMemory.memory.Forget(ctx, ids)
}

/*
Close closes the projector and wrapped memory.
*/
func (projectedMemory *ProjectedMemory) Close() error {
	projectedMemory.cancel()

	if err := projectedMemory.projector.Close(); err != nil {
		return err
	}

	return projectedMemory.memory.Close()
}

func (projectedMemory *ProjectedMemory) project(
	ctx context.Context,
	consolidation MemoryConsolidation,
) (MemoryConsolidation, error) {
	if err := consolidation.Validate(); err != nil {
		return MemoryConsolidation{}, err
	}

	if len(consolidation.Records) == 0 {
		return consolidation, nil
	}

	texts := make([]string, len(consolidation.Records))
	for index, record := range consolidation.Records {
		texts[index] = record.Text
	}

	projections, err := projectedMemory.projector.ProjectBatch(ctx, texts)
	if err != nil {
		return MemoryConsolidation{}, err
	}

	if len(projections) != len(consolidation.Records) {
		return MemoryConsolidation{}, errnie.Err(
			errnie.Validation,
			"memory projection count mismatch",
			nil,
		)
	}

	enriched := consolidation
	enriched.Records = append([]MemoryRecord(nil), consolidation.Records...)

	for index, projection := range projections {
		enriched.Records[index].Embedding = append([]float32(nil), projection.Embedding...)
	}

	return enriched, nil
}
