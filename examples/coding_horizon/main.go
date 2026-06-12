// Long-horizon AI-native coding workflow for a target repository.
//
// This example is optimized for agent strengths (narrow context, structured plans,
// machine verification) and weaknesses (hallucination, unbounded diffs, weak done-detection).
//
// Run from the repository root:
//
//	ANIMAL_AGENT_WORKSPACE=/path/to/repo \
//	  go run -ldflags=-checklinkname=0 ./examples/coding_horizon \
//	  -goal "Add retry logic to the HTTP client"
//
// Hygiene-only mode (no goal):
//
//	ANIMAL_AGENT_WORKSPACE=/path/to/repo \
//	  go run -ldflags=-checklinkname=0 ./examples/coding_horizon
//
// Deterministic dry-run (observe, backlog, verify; no LLM edits):
//
//	make example-coding-horizon-dry
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/theapemachine/animal/examples/support"
)

func main() {
	goal := flag.String("goal", "", "high-level outcome to achieve; omit for hygiene-only horizon")
	workspace := flag.String("workspace", os.Getenv("ANIMAL_AGENT_WORKSPACE"), "repository root to analyze and edit")
	maxCycles := flag.Int("max-cycles", 24, "maximum recon/plan/mutate/prove cycles")
	dryRun := flag.Bool("dry-run", false, "skip LLM mutation; run observe, backlog, and verify only")
	flag.Parse()

	if loadErr := support.LoadViper(); loadErr != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", loadErr)
		os.Exit(1)
	}

	if *workspace == "" {
		fmt.Fprintln(os.Stderr, "workspace is required via -workspace or ANIMAL_AGENT_WORKSPACE")
		os.Exit(1)
	}

	ctx := context.Background()
	pool := support.NewQPool(ctx)

	orchestrator, err := newOrchestrator(ctx, pool, Config{
		Workspace: *workspace,
		Goal:      *goal,
		MaxCycles: *maxCycles,
		DryRun:    *dryRun,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "orchestrator: %v\n", err)
		os.Exit(1)
	}

	defer orchestrator.Close()

	if runErr := orchestrator.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "run: %v\n", runErr)
		os.Exit(1)
	}
}
