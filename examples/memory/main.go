package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/theapemachine/errnie"
)

const (
	cognitiveScope  = "goal-memory-cognitive-example"
	frictionScope   = "goal-memory-friction-example"
	analogScope     = "goal-memory-analog-example"
	projectionScope = "goal-memory-projection-example"
	manifoldScope   = "goal-memory-manifold-example"

	cognitiveRecallLimit  = 8
	frictionRecallLimit   = 8
	analogRecallLimit     = 8
	projectionRecallLimit = 8
	manifoldRecallLimit   = 8

	manifoldInputDim  = 16
	manifoldHiddenDim = 8
	manifoldLatentDim = 4
	manifoldBatch     = 2
	manifoldAlpha     = 0.05
)

/*
Demo runs memory examples with deterministic output.
*/
type Demo struct {
	ctx    context.Context
	cancel context.CancelFunc
	output io.Writer
}

/*
NewDemo instantiates a memory example runner.
*/
func NewDemo(ctx context.Context, output io.Writer) (*Demo, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "memory example context is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	demo := &Demo{
		ctx:    ctx,
		cancel: cancel,
		output: output,
	}

	return demo, errnie.Require(map[string]any{
		"ctx":    demo.ctx,
		"cancel": demo.cancel,
		"output": demo.output,
	})
}

/*
Close cancels the example scope.
*/
func (demo *Demo) Close() {
	demo.cancel()
}

func main() {
	mode := flag.String("mode", "dmt", "memory example mode")
	flag.Parse()

	demo, err := NewDemo(context.Background(), os.Stdout)
	if err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}

	switch *mode {
	case "dmt":
		err = demo.Cognitive()
	case "dmt-friction":
		err = demo.Friction()
	case "dmt-analog":
		err = demo.Analog()
	case "projection":
		err = demo.Projection()
	case "manifold":
		err = demo.Manifold()
	default:
		err = errnie.Err(
			errnie.Validation,
			"memory example mode must be dmt, dmt-friction, dmt-analog, projection, or manifold",
			nil,
		)
	}

	if err != nil {
		printError(os.Stderr, err)
		os.Exit(1)
	}
}

func printError(output io.Writer, err error) {
	fmt.Fprintf(output, "memory example: %v\n", err)

	for cause := errors.Unwrap(err); cause != nil; cause = errors.Unwrap(cause) {
		fmt.Fprintf(output, "caused by: %v\n", cause)
	}
}
