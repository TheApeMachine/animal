// Two-phase demo: gossip merge via Agent.Cycle(), then an LLM reply when a provider is up.
//
// Phase 1 always runs (fast, no network).
// Phase 2 calls ai.endpoint from cmd/cfg/config.yml (default http://localhost:1234/v1).
//
// Run from the repository root:
//
//	make example-swarm-agent-cycle
//
// For phase 2, start a local OpenAI-compatible server (e.g. LM Studio) and set OPENAI_API_KEY if required.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/examples/support"
	"github.com/theapemachine/animal/internal"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

func main() {
	if loadErr := support.LoadViper(); loadErr != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", loadErr)
		os.Exit(1)
	}

	ctx := context.Background()
	pool := support.NewQPool(ctx)

	registry, err := swarm.NewRegistry(
		ctx, pool,
		support.DefaultSwarmOptions("example-agent-cycle"),
		support.DefaultLeaseOptions(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "registry: %v\n", err)
		os.Exit(1)
	}

	developer, err := ai.NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/vertical-a/"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "developer: %v\n", err)
		os.Exit(1)
	}

	peer, err := registry.NewParticipant("peer-pm", "Morgan", "project_manager", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "peer: %v\n", err)
		os.Exit(1)
	}

	topic := "roadmap.announce"
	payload := "ship swarm gossip before orchestrator workflows"

	if announceErr := peer.Announce(topic, payload); announceErr != nil {
		fmt.Fprintf(os.Stderr, "announce: %v\n", announceErr)
		os.Exit(1)
	}

	record, err := waitForAnnounce(developer, topic, 2*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gossip: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- phase 1: gossip (no LLM) ---")
	fmt.Printf("agent %s merged announce %q from %s\n", developer.Name, record.Topic, record.ActorName)
	fmt.Printf("payload: %s\n", record.Payload)

	developer.Context.Messages = append(developer.Context.Messages, provider.Message{
		Role: "user",
		Content: fmt.Sprintf(
			"%s (%s) announced over the swarm mesh:\n\n%s\n\nAcknowledge in one sentence and state your next build step for lanes/vertical-a/.",
			record.ActorName, record.Role, record.Payload,
		),
	})

	fmt.Println("--- phase 2: LLM response ---")

	endpoint, apiKey, model := support.OpenAIConfig()

	llm, err := provider.NewOpenAI(ctx, pool, endpoint, apiKey, model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "provider config: %v\n", err)
		os.Exit(1)
	}

	consumer := pool.CreateBroadcastGroup(internal.ChannelMessages.String()).Acquire(
		"example-agent-cycle-stream",
		nil,
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				artifact, waitErr := consumer.Wait(ctx)

				if waitErr != nil {
					fmt.Fprintf(os.Stderr, "consumer wait: %v\n", waitErr)
					return
				}

				chunk, decodeErr := qpool.ArtifactValue[string](artifact)

				if decodeErr != nil {
					fmt.Fprintf(os.Stderr, "consumer decode: %v\n", decodeErr)
					continue
				}

				fmt.Printf("Ada: %s\n", chunk)
			}
		}
	}()

	err = llm.Stream(developer.System, &developer.Context, provider.NewParams())
}

func waitForAnnounce(developer *ai.Agent, topic string, timeout time.Duration) (swarm.AnnounceRecord, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		developer.Cycle()

		for _, record := range developer.Participant().View().RecentAnnounces() {
			if record.Topic == topic {
				return record, nil
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	return swarm.AnnounceRecord{}, fmt.Errorf("timed out waiting for announce topic %q", topic)
}
