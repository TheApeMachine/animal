package ai

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"unicode"

	"github.com/theapemachine/errnie"
	"github.com/theapemachine/nomagique/learning/manifold"
)

/*
ManifoldConfig controls the nomagique resonance projector.
*/
type ManifoldConfig struct {
	Architecture []int
	TargetDim    int
	Batch        int
	Alpha        float64
}

/*
ManifoldProjector projects memory text through nomagique's resonance manifold.
*/
type ManifoldProjector struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	config ManifoldConfig
	solver *manifold.BatchSolver
	mutex  sync.Mutex
}

/*
NewManifoldProjector instantiates a nomagique-backed memory projector.
*/
func NewManifoldProjector(
	ctx context.Context,
	config ManifoldConfig,
) (*ManifoldProjector, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "manifold projector context is required", nil)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	solver, err := manifold.NewBatchSolver(
		append([]int(nil), config.Architecture...),
		config.TargetDim,
		config.Batch,
		config.Alpha,
	)
	if err != nil {
		cancel()

		return nil, errnie.Err(errnie.Validation, "manifold projector creation failed", err)
	}

	projector := &ManifoldProjector{
		ctx:    ctx,
		cancel: cancel,
		config: config,
		solver: solver,
	}

	return projector, errnie.Require(map[string]any{
		"ctx":    projector.ctx,
		"cancel": projector.cancel,
		"solver": projector.solver,
	})
}

/*
Validate checks manifold projector configuration.
*/
func (config ManifoldConfig) Validate() error {
	if len(config.Architecture) < 2 {
		return errnie.Err(errnie.Validation, "manifold architecture is required", nil)
	}

	for _, dimension := range config.Architecture {
		if dimension <= 0 {
			return errnie.Err(errnie.Validation, "manifold architecture dimensions must be positive", nil)
		}
	}

	if config.Batch <= 0 {
		return errnie.Err(errnie.Validation, "manifold batch is required", nil)
	}

	if config.Alpha <= 0 || config.Alpha > 1 {
		return errnie.Err(errnie.Validation, "manifold alpha must be between 0 and 1", nil)
	}

	return nil
}

/*
Project projects one text sample.
*/
func (projector *ManifoldProjector) Project(
	ctx context.Context,
	text string,
) (MemoryProjection, error) {
	projections, err := projector.ProjectBatch(ctx, []string{text})
	if err != nil {
		return MemoryProjection{}, err
	}

	return projections[0], nil
}

/*
ProjectBatch projects text samples in manifold batches.
*/
func (projector *ManifoldProjector) ProjectBatch(
	ctx context.Context,
	texts []string,
) ([]MemoryProjection, error) {
	if err := projector.validate(ctx, texts); err != nil {
		return nil, err
	}

	projector.mutex.Lock()
	defer projector.mutex.Unlock()

	projections := make([]MemoryProjection, 0, len(texts))

	for start := 0; start < len(texts); start += projector.config.Batch {
		end := min(start+projector.config.Batch, len(texts))
		chunk, err := projector.projectChunk(texts[start:end])
		if err != nil {
			return nil, err
		}

		projections = append(projections, chunk...)
	}

	return projections, nil
}

/*
Close closes the manifold solver.
*/
func (projector *ManifoldProjector) Close() error {
	projector.cancel()
	projector.solver.Close()

	return nil
}

func (projector *ManifoldProjector) validate(
	ctx context.Context,
	texts []string,
) error {
	if ctx == nil {
		return errnie.Err(errnie.Validation, "manifold projection context is required", nil)
	}

	if projector.ctx.Err() != nil {
		return errnie.Err(errnie.Timeout, "manifold projector is closed", projector.ctx.Err())
	}

	if len(texts) == 0 {
		return errnie.Err(errnie.Validation, "manifold projection text is required", nil)
	}

	for _, text := range texts {
		if err := ensureCognitiveText(text); err != nil {
			return err
		}
	}

	return nil
}

func (projector *ManifoldProjector) projectChunk(
	texts []string,
) ([]MemoryProjection, error) {
	inputs := projector.inputs(texts)

	if err := projector.solver.SetInputs(inputs, nil); err != nil {
		return nil, errnie.Err(errnie.Validation, "manifold inputs failed", err)
	}

	if err := projector.solver.Settle(true); err != nil {
		return nil, errnie.Err(errnie.Validation, "manifold settle failed", err)
	}

	if err := projector.solver.ReadOutcomes(); err != nil {
		return nil, errnie.Err(errnie.Validation, "manifold outcome read failed", err)
	}

	return projector.outcomes(len(texts))
}

func (projector *ManifoldProjector) inputs(texts []string) []float64 {
	inputDim := projector.config.Architecture[0]
	inputs := make([]float64, projector.config.Batch*inputDim)

	for index, text := range texts {
		copy(inputs[index*inputDim:(index+1)*inputDim], manifoldInput(text, inputDim))
	}

	return inputs
}

func (projector *ManifoldProjector) outcomes(count int) ([]MemoryProjection, error) {
	projections := make([]MemoryProjection, 0, count)

	for slot := range count {
		latent, energy, surprise, err := projector.solver.OutcomeSlot(slot)
		if err != nil {
			return nil, errnie.Err(errnie.Validation, "manifold outcome failed", err)
		}

		projections = append(projections, MemoryProjection{
			Embedding: float32Vector(latent),
			Energy:    energy,
			Surprise:  surprise,
		})
	}

	return projections, nil
}

func manifoldInput(text string, dimension int) []float64 {
	vector := make([]float64, dimension)

	for _, token := range manifoldTokens(text) {
		hash := fnv.New64a()
		_, _ = hash.Write([]byte(token))
		vector[int(hash.Sum64()%uint64(dimension))]++
	}

	return normalizeVector(vector)
}

func manifoldTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}

func normalizeVector(vector []float64) []float64 {
	sumSquares := 0.0

	for _, value := range vector {
		sumSquares += value * value
	}

	if sumSquares == 0 {
		return vector
	}

	scale := 1.0 / math.Sqrt(sumSquares)
	for index := range vector {
		vector[index] *= scale
	}

	return vector
}

func float32Vector(values []float64) []float32 {
	out := make([]float32, len(values))

	for index, value := range values {
		out[index] = float32(value)
	}

	return out
}
