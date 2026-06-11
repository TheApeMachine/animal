package conversation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
)

/*
Panel runs an endless salon where alignment clusters emerge from stance themes.
*/
type Panel struct {
	ctx             context.Context
	llm             *provider.OpenAI
	registry        *SalonRegistry
	speakers        []Speaker
	names           map[string]string
	openingQuestion string
	turnCount       int
}

/*
Speaker binds a configured persona to a swarm-attached agent.
*/
type Speaker struct {
	Persona string
	Agent   *ai.Agent
}

/*
NewPanel wires speakers to a shared salon registry.
*/
func NewPanel(
	ctx context.Context,
	llm *provider.OpenAI,
	registry *SalonRegistry,
	speakers []Speaker,
) (*Panel, error) {
	if registry == nil {
		return nil, fmt.Errorf("conversation: salon registry is required")
	}

	if len(speakers) == 0 {
		return nil, fmt.Errorf("conversation: at least one speaker is required")
	}

	if llm == nil {
		return nil, fmt.Errorf("conversation: LLM provider is required")
	}

	names := make(map[string]string, len(speakers))

	for _, speaker := range speakers {
		if speaker.Agent == nil {
			return nil, fmt.Errorf("conversation: speaker agent is required")
		}

		names[speaker.Agent.ID] = speaker.Agent.Name
	}

	return &Panel{
		ctx:      ctx,
		llm:      llm,
		registry: registry,
		speakers: speakers,
		names:    names,
	}, nil
}

/*
Bootstrap stores the opening question for the first reply.
*/
func (panel *Panel) Bootstrap(seed string) error {
	panel.openingQuestion = strings.TrimSpace(seed)
	return nil
}

/*
Run loops until context cancellation, rotating speakers each round.
*/
func (panel *Panel) Run() error {
	fmt.Println("salon conversation running (Ctrl+C to stop)")

	if panel.openingQuestion == "" {
		return fmt.Errorf("conversation: opening question is required")
	}

	if err := panel.syncGossip(); err != nil {
		return err
	}

	round := 0

	for panel.ctx.Err() == nil {
		for _, speaker := range panel.speakers {
			if panel.ctx.Err() != nil {
				return nil
			}

			if err := panel.syncGossip(); err != nil {
				return err
			}

			spoken, themes, err := panel.speak(speaker)
			if err != nil {
				if panel.ctx.Err() != nil || errors.Is(err, context.Canceled) {
					return nil
				}

				fmt.Fprintf(os.Stderr, "speak: %v\n", err)
				continue
			}

			if err := PublishTurn(speaker.Agent.Participant(), spoken); err != nil {
				return err
			}

			if err := PublishStance(speaker.Agent.Participant(), themes); err != nil {
				return err
			}

			if err := panel.syncGossip(); err != nil {
				return err
			}

			fmt.Printf("%s: %s\n", speaker.Agent.Name, spoken)

			panel.turnCount++
		}

		round++
		clusters := ComputeClusters(panel.registry, panel.names, minClusterOverlap)
		fmt.Println(FormatClusters(clusters))
		fmt.Printf("--- end of round %d ---\n\n", round)
	}

	return nil
}

func (panel *Panel) speak(speaker Speaker) (string, []string, error) {
	speaker.Agent.Context.Messages = BuildReplyMessages(
		panel.registry.Turns(),
		panel.openingQuestion,
		speaker.Agent.Name,
	)

	reply, err := collectStreamReply(panel.ctx, panel.llm, speaker.Agent.System, &speaker.Agent.Context)
	if err != nil {
		if panel.ctx.Err() != nil || errors.Is(err, context.Canceled) {
			return "", nil, context.Canceled
		}

		return "", nil, err
	}

	spoken := strings.TrimSpace(reply)

	return spoken, ThemesFromContent(spoken), nil
}

func (panel *Panel) syncGossip() error {
	for _, speaker := range panel.speakers {
		speaker.Agent.Cycle()

		if drainErr := speaker.Agent.Participant().Drain(); drainErr != nil {
			return drainErr
		}

		for _, record := range speaker.Agent.Participant().View().RecentAnnounces() {
			if err := panel.registry.Apply(record); err != nil {
				return err
			}
		}
	}

	return nil
}
