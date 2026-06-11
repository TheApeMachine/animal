// Parallel builders claim exclusive lane prefixes through gossip and leases.
//
// Run from the repository root:
//
//	make example-swarm-parallel-claims
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/theapemachine/animal/examples/support"
	"github.com/theapemachine/animal/swarm"
)

func main() {
	ctx := context.Background()
	pool := support.NewQPool(ctx)

	registry, err := swarm.NewRegistry(
		ctx, pool,
		support.DefaultSwarmOptions("example-parallel-claims"),
		support.DefaultLeaseOptions(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(1)
	}

	builderA, err := registry.NewParticipant("builder-a", "Ada", "developer", []string{"lanes/vertical-a/"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "participant a: %v\n", err)
		os.Exit(1)
	}

	builderB, err := registry.NewParticipant("builder-b", "Bob", "developer", []string{"lanes/vertical-b/"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "participant b: %v\n", err)
		os.Exit(1)
	}

	observer, err := registry.NewParticipant("reviewer", "Quinn", "reviewer", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "observer: %v\n", err)
		os.Exit(1)
	}

	prefixA, err := builderA.TryClaimConfigured()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claim a: %v\n", err)
		os.Exit(1)
	}

	prefixB, err := builderB.TryClaimConfigured()
	if err != nil {
		fmt.Fprintf(os.Stderr, "claim b: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(20 * time.Millisecond)
	if drainErr := support.DrainParticipant(observer); drainErr != nil {
		fmt.Fprintf(os.Stderr, "drain: %v\n", drainErr)
		os.Exit(1)
	}

	view := observer.View()
	holderA, okA := view.ClaimHolder("lanes/vertical-a/")
	holderB, okB := view.ClaimHolder("lanes/vertical-b/")

	fmt.Printf("builder-a claimed %s\n", prefixA)
	fmt.Printf("builder-b claimed %s\n", prefixB)
	fmt.Printf("observer sees vertical-a held by %q (ok=%v)\n", holderA, okA)
	fmt.Printf("observer sees vertical-b held by %q (ok=%v)\n", holderB, okB)

	conflictErr := builderB.TryClaim("lanes/vertical-a/")
	if conflictErr == nil {
		fmt.Fprintln(os.Stderr, "expected lease conflict on vertical-a")
		os.Exit(1)
	}

	fmt.Printf("conflicting claim rejected: %v\n", conflictErr)

	if releaseErr := builderA.Release(prefixA); releaseErr != nil {
		fmt.Fprintf(os.Stderr, "release: %v\n", releaseErr)
		os.Exit(1)
	}

	if releaseErr := builderB.Release(prefixB); releaseErr != nil {
		fmt.Fprintf(os.Stderr, "release: %v\n", releaseErr)
		os.Exit(1)
	}

	fmt.Println("leases released; gossip updated")
}
