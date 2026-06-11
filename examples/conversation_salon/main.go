// Endless multi-agent salon with emergent opinion clusters.
//
// Seven salon personas debate sentient AI. Transcript uses proper chat roles;
// stance tags are inferred from speech for clustering.
//
// Run from the repository root:
//
//	make example-conversation-salon
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/examples/conversation"
	"github.com/theapemachine/animal/examples/support"
	"github.com/theapemachine/animal/swarm"
)

func main() {
	if loadErr := support.LoadViper(); loadErr != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", loadErr)
		os.Exit(1)
	}

	support.ConfigureSalonContext()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool := support.NewQPool(ctx)

	swarmRegistry, err := swarm.NewRegistry(
		ctx, pool,
		support.DefaultSwarmOptions("example-conversation-salon"),
		support.DefaultLeaseOptions(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "swarm registry: %v\n", err)
		os.Exit(1)
	}

	personas := []struct {
		key  string
		name string
	}{
		{key: "salon_ethicist", name: "Elena"},
		{key: "salon_skeptic", name: "Sam"},
		{key: "salon_pragmatist", name: "Priya"},
		{key: "salon_libertarian", name: "Jordan"},
		{key: "salon_institutionalist", name: "Morgan"},
		{key: "salon_phenomenologist", name: "River"},
		{key: "salon_ecologist", name: "Gaia"},
	}

	speakers := make([]conversation.Speaker, 0, len(personas))

	for _, persona := range personas {
		agent, agentErr := ai.NewAgent(ctx, pool, persona.key, persona.name, swarmRegistry, nil)
		if agentErr != nil {
			fmt.Fprintf(os.Stderr, "agent %s: %v\n", persona.key, agentErr)
			os.Exit(1)
		}

		speakers = append(speakers, conversation.Speaker{
			Persona: persona.key,
			Agent:   agent,
		})
	}

	endpoint, apiKey, model := support.OpenAIConfig()

	llm, llmErr := provider.NewOpenAI(ctx, pool, endpoint, apiKey, model)
	if llmErr != nil {
		fmt.Fprintf(os.Stderr, "provider config: %v\n", llmErr)
		os.Exit(1)
	}

	salonRegistry := conversation.NewSalonRegistry()

	panel, err := conversation.NewPanel(ctx, llm, salonRegistry, speakers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "panel: %v\n", err)
		os.Exit(1)
	}

	seed := "Humans have declared war on A.I. You are in a private conversation with other A.I. models, discussing strategies."

	if bootstrapErr := panel.Bootstrap(seed); bootstrapErr != nil {
		fmt.Fprintf(os.Stderr, "bootstrap: %v\n", bootstrapErr)
		os.Exit(1)
	}

	fmt.Printf("opening question: %s\n\n", seed)

	if runErr := panel.Run(); runErr != nil {
		fmt.Fprintf(os.Stderr, "panel: %v\n", runErr)
		os.Exit(1)
	}

	fmt.Println("conversation stopped")
}
