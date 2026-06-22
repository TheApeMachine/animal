package main

import (
	"context"
	"strings"
	"unicode"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/errnie"
)

const signalEmbeddingCap = 16.0

/*
SignalProjector projects text into compact memory quality signals.
*/
type SignalProjector struct {
	ctx    context.Context
	cancel context.CancelFunc
}

/*
NewSignalProjector instantiates a deterministic example projector.
*/
func NewSignalProjector(ctx context.Context) (*SignalProjector, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "signal projector context is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	projector := &SignalProjector{
		ctx:    ctx,
		cancel: cancel,
	}

	return projector, errnie.Require(map[string]any{
		"ctx":    projector.ctx,
		"cancel": projector.cancel,
	})
}

/*
Project projects one text sample.
*/
func (projector *SignalProjector) Project(
	ctx context.Context,
	text string,
) (ai.MemoryProjection, error) {
	projections, err := projector.ProjectBatch(ctx, []string{text})
	if err != nil {
		return ai.MemoryProjection{}, err
	}

	return projections[0], nil
}

/*
ProjectBatch projects text samples into density, novelty, and signal ratios.
*/
func (projector *SignalProjector) ProjectBatch(
	ctx context.Context,
	texts []string,
) ([]ai.MemoryProjection, error) {
	if err := projector.validate(ctx, texts); err != nil {
		return nil, err
	}

	projections := make([]ai.MemoryProjection, 0, len(texts))

	for _, text := range texts {
		projection, err := projector.project(text)
		if err != nil {
			return nil, err
		}

		projections = append(projections, projection)
	}

	return projections, nil
}

/*
Close closes the projector scope.
*/
func (projector *SignalProjector) Close() error {
	projector.cancel()

	return nil
}

func (projector *SignalProjector) validate(ctx context.Context, texts []string) error {
	if ctx == nil {
		return errnie.Err(errnie.Validation, "signal projection context is required", nil)
	}

	if projector.ctx.Err() != nil {
		return errnie.Err(errnie.Timeout, "signal projector is closed", projector.ctx.Err())
	}

	if len(texts) == 0 {
		return errnie.Err(errnie.Validation, "signal projection text is required", nil)
	}

	return nil
}

func (projector *SignalProjector) project(text string) (ai.MemoryProjection, error) {
	tokens := signalTokens(text)
	if len(tokens) == 0 {
		return ai.MemoryProjection{}, errnie.Err(
			errnie.Validation,
			"signal projection text is required",
			nil,
		)
	}

	unique := map[string]struct{}{}
	signals := 0

	for _, token := range tokens {
		unique[token] = struct{}{}

		if isSignalToken(token) {
			signals++
		}
	}

	tokenCount := float64(len(tokens))
	density := tokenCount / signalEmbeddingCap
	if density > 1 {
		density = 1
	}

	novelty := float64(len(unique)) / tokenCount
	signalRatio := float64(signals) / tokenCount

	return ai.MemoryProjection{
		Embedding: []float32{
			float32(density),
			float32(novelty),
			float32(signalRatio),
		},
		Energy:   density,
		Surprise: novelty,
	}, nil
}

func signalTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}

func isSignalToken(token string) bool {
	switch token {
	case "blocker", "drift", "friction", "goal", "quality", "signal", "signals":
		return true
	}

	return false
}
