// Exclusive path-prefix leases gate workspace writes and surface advisory reads.
//
// Run from the repository root:
//
//	make example-lease-workspace
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theapemachine/animal/ai/tool/editor/doc"
	"github.com/theapemachine/animal/ai/tool/editor/fs"
	"github.com/theapemachine/animal/examples/support"
	"github.com/theapemachine/animal/lease"
)

func main() {
	ctx := context.Background()

	workDir, err := os.MkdirTemp("", "animal-lease-example-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(workDir)

	laneA := filepath.Join(workDir, "lanes", "vertical-a")
	if mkdirErr := os.MkdirAll(laneA, 0o755); mkdirErr != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", mkdirErr)
		os.Exit(1)
	}

	samplePath := filepath.Join(laneA, "main.go")
	if writeErr := os.WriteFile(samplePath, []byte("package main\n"), 0o644); writeErr != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", writeErr)
		os.Exit(1)
	}

	coordinator, err := lease.NewCoordinator(support.DefaultLeaseOptions())
	if err != nil {
		fmt.Fprintf(os.Stderr, "coordinator: %v\n", err)
		os.Exit(1)
	}

	document, err := fs.NewDocument(workDir, coordinator)
	if err != nil {
		fmt.Fprintf(os.Stderr, "document: %v\n", err)
		os.Exit(1)
	}

	builderA := lease.Principal{ActorID: "builder-a", RequireLease: true}
	builderB := lease.Principal{ActorID: "builder-b", RequireLease: true}
	reviewer := lease.Principal{ActorID: "reviewer", ReadOnly: true}

	relativePath := "lanes/vertical-a/main.go"

	readErr := coordinator.ObserveRead(relativePath, reviewer)
	if readErr != nil {
		fmt.Fprintf(os.Stderr, "unexpected read advisory before leases: %v\n", readErr)
		os.Exit(1)
	}

	fmt.Println("reviewer can read before any lease is held")

	if acquireErr := coordinator.AcquireID("lanes/vertical-a/", "builder-a"); acquireErr != nil {
		fmt.Fprintf(os.Stderr, "acquire: %v\n", acquireErr)
		os.Exit(1)
	}

	changingErr := coordinator.ObserveRead(relativePath, reviewer)
	changing, isChanging := lease.AsChanging(changingErr)

	if !isChanging {
		fmt.Fprintln(os.Stderr, "expected advisory ChangingError while builder-a holds the lease")
		os.Exit(1)
	}

	fmt.Printf("reviewer sees changing advisory: held by %q on prefix %q\n", changing.ActorID, changing.LeaseKey)

	builderBWriteErr := coordinator.CanWrite(relativePath, builderB)
	if builderBWriteErr == nil {
		fmt.Fprintln(os.Stderr, "expected builder-b write to fail without a lease")
		os.Exit(1)
	}

	fmt.Printf("builder-b write blocked: %v\n", builderBWriteErr)

	if writeErr := coordinator.CanWrite(relativePath, builderA); writeErr != nil {
		fmt.Fprintf(os.Stderr, "builder-a write: %v\n", writeErr)
		os.Exit(1)
	}

	replaceErr := document.Replace(ctx, doc.ReplaceParams{
		Path: relativePath,
		Old:  "package main",
		New:  "package verticala",
	})
	if replaceErr != nil {
		fmt.Fprintf(os.Stderr, "replace: %v\n", replaceErr)
		os.Exit(1)
	}

	readResult, readFileErr := document.Read(ctx, doc.ReadParams{Path: relativePath})
	if readFileErr != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", readFileErr)
		os.Exit(1)
	}

	fmt.Printf("builder-a replaced file; content now:\n%s\n", readResult.Content)

	if releaseErr := coordinator.ReleaseID("lanes/vertical-a/", "builder-a"); releaseErr != nil {
		fmt.Fprintf(os.Stderr, "release: %v\n", releaseErr)
		os.Exit(1)
	}

	fmt.Println("lease released; reviewer can read stable content again")
}
