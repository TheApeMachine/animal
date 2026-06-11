// A project manager announces roadmap intent; a developer merges the gossip locally.
//
// Run from the repository root:
//
//	make example-swarm-roadmap-gossip
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
		support.DefaultSwarmOptions("example-roadmap-gossip"),
		support.DefaultLeaseOptions(),
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(1)
	}

	manager, err := registry.NewParticipant("pm-1", "Morgan", "project_manager", nil)
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "manager: %v\n", err)
		os.Exit(1)
	}

	developer, err := registry.NewParticipant("dev-1", "Ada", "developer", []string{"lanes/vertical-a/"})
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "developer: %v\n", err)
		os.Exit(1)
	}

	topic := "roadmap.announce"
	payload := "prioritize lease-backed parallel lanes for vertical-a and vertical-b"

	if announceErr := manager.Announce(topic, payload); announceErr != nil {
		fmt.Fprintf(os.Stderr, "announce: %v\n", announceErr)
		os.Exit(1)
	}

	if waitErr := support.WaitAnnounce(developer, topic, 2*time.Second); waitErr != nil {
		fmt.Fprintf(os.Stderr, "wait: %v\n", waitErr)
		os.Exit(1)
	}

	records := developer.View().RecentAnnounces()
	
	if len(records) != 1 {
		fmt.Fprintf(os.Stderr, "expected 1 announce, got %d\n", len(records))
		os.Exit(1)
	}

	record := records[0]
	fmt.Printf("developer heard %q from %s (%s)\n", record.Topic, record.ActorName, record.Role)
	fmt.Printf("payload: %s\n", record.Payload)

	if statusErr := developer.PublishStatus("ready"); statusErr != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", statusErr)
		os.Exit(1)
	}

	fmt.Println("developer published status=ready on the mesh")
}
