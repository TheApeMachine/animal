package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
)

/*
TrainingSink records successful agent traces for future fine-tuning.
*/
type TrainingSink interface {
	Record(agent *Agent, metric swarm.Metric) error
}

/*
TrainingRecorder appends successful agent traces as chat fine-tuning JSONL.
*/
type TrainingRecorder struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	path   string
}

/*
NewTrainingRecorder instantiates a JSONL recorder.
*/
func NewTrainingRecorder(ctx context.Context, path string) (*TrainingRecorder, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "training recorder context is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	if strings.TrimSpace(path) == "" {
		cancel()
		return nil, errnie.Err(errnie.Validation, "training recorder path is required", nil)
	}

	trainingRecorder := &TrainingRecorder{
		ctx:    ctx,
		cancel: cancel,
		path:   filepath.Clean(path),
	}

	return trainingRecorder, errnie.Require(map[string]any{
		"ctx":    trainingRecorder.ctx,
		"cancel": trainingRecorder.cancel,
		"path":   trainingRecorder.path,
	})
}

/*
Record appends the agent trace when metric marks the outcome successful.
*/
func (trainingRecorder *TrainingRecorder) Record(
	agent *Agent,
	metric swarm.Metric,
) error {
	if agent == nil {
		return errnie.Err(errnie.Validation, "training record agent is required", nil)
	}

	if err := metric.Validate(); err != nil {
		return err
	}

	if !metric.Success {
		return errnie.Err(errnie.Validation, "training metric must be successful", nil)
	}

	example, err := agent.FineTuneExample(metric)
	if err != nil {
		return err
	}

	return trainingRecorder.RecordExample(example)
}

/*
RecordExample appends one validated JSONL fine-tuning record.
*/
func (trainingRecorder *TrainingRecorder) RecordExample(example FineTuneExample) error {
	payload, err := example.JSONL()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(trainingRecorder.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return errnie.Err(errnie.IO, "training jsonl open failed", err)
	}

	defer file.Close()

	if _, err := file.Write(payload); err != nil {
		return errnie.Err(errnie.IO, "training jsonl write failed", err)
	}

	return nil
}
