package ai

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Agent is one configured LLM participant with persona, tools, memory, and optional swarm attachment.
It polls incoming gossip or coop traffic and appends provider messages into a shared conversation context.
*/
type Agent struct {
	ctx           context.Context
	cancel        context.CancelFunc
	err           error
	pool          *qpool.Q[any]
	ID            string
	Name          string
	Role          string
	System        string
	Tools         []Tool
	Memory        []Memory
	Context       provider.Context
	participant   *swarm.Participant
	coopBroadcast *qpool.BroadcastGroup
	coopChannel   *qpool.BroadcastConsumer
}

func NewAgent(
	ctx context.Context,
	pool *qpool.Q[any],
	role, name string,
	registry *swarm.Registry,
	claimPrefixes []string,
) (*Agent, error) {
	ctx, cancel := context.WithCancel(ctx)

	template := viper.GetViper().GetString("ai.prompt.template.system")

	system := strings.ReplaceAll(template, "{{ agent.role }}", role)
	system = strings.ReplaceAll(system, "{{ agent.name }}", name)
	system = strings.ReplaceAll(system, "{{ project.name }}", viper.GetViper().GetString("project.name"))
	system = strings.ReplaceAll(system, "{{ project.description }}", viper.GetViper().GetString("project.description"))
	system = strings.ReplaceAll(system, "{{ agent.characteristics }}", personaLinesFromViper(role, "characteristics"))
	system = strings.ReplaceAll(system, "{{ agent.responsibilities }}", personaLinesFromViper(role, "responsibilities"))
	system = strings.ReplaceAll(system, "{{ agent.guidelines }}", personaLinesFromViper(role, "guidelines"))

	agent := &Agent{
		ctx:     ctx,
		cancel:  cancel,
		pool:    pool,
		ID:      uuid.New().String(),
		Name:    name,
		Role:    role,
		System:  strings.TrimSpace(system),
		Tools:   make([]Tool, 0),
		Memory:  make([]Memory, 0),
		Context: *provider.NewContext(ctx),
	}

	if registry != nil {
		participant, participantErr := registry.NewParticipant(agent.ID, name, role, claimPrefixes)

		if participantErr != nil {
			return nil, participantErr
		}

		agent.participant = participant
	} else {
		agent.coopBroadcast = pool.CreateBroadcastGroup("coop", 64)
		agent.coopChannel = agent.coopBroadcast.Subscribe(agent.ID, 64)
	}

	return agent, errnie.Require(map[string]any{
		"ctx":     agent.ctx,
		"cancel":  agent.cancel,
		"pool":    agent.pool,
		"ID":      agent.ID,
		"Name":    agent.Name,
		"Role":    agent.Role,
		"System":  agent.System,
		"Tools":   agent.Tools,
		"Memory":  agent.Memory,
		"Context": agent.Context,
	})
}

func (agent *Agent) Cycle() {
	for {
		select {
		case <-agent.ctx.Done():
			return
		default:
		}

		qv := agent.pollIncoming()
		if qv != nil {
			agent.handleIncoming(qv)
			continue
		}

		if agent.participant != nil {
			if drainErr := agent.participant.Drain(); drainErr != nil {
				errnie.Warn("swarm drain failed", "err", drainErr)
				return
			}
		}

		return
	}
}

func (agent *Agent) pollIncoming() *qpool.QValue[any] {
	if agent.participant != nil {
		return agent.participant.Poll()
	}

	if agent.coopChannel == nil {
		return nil
	}

	return agent.coopChannel.Poll()
}

func (agent *Agent) handleIncoming(qv *qpool.QValue[any]) {
	switch payload := qv.Value.(type) {
	case swarm.Rumor:
		if agent.participant == nil {
			errnie.Warn("agent received swarm rumor without participant")
			return
		}

		if receiveErr := agent.participant.Receive(payload); receiveErr != nil {
			errnie.Warn("swarm receive failed", "err", receiveErr)
		}
	case provider.Message:
		agent.Context.Messages = append(agent.Context.Messages, payload)
	default:
		errnie.Warn("agent channel received invalid payload type", "type", qv.Value)
	}
}

/*
Participant returns the agent's swarm coordination surface when attached.
*/
func (agent *Agent) Participant() *swarm.Participant {
	return agent.participant
}

func personaLinesFromViper(personaKey, field string) string {
	configPath := "ai.personas." + personaKey + "." + field
	values := viper.GetStringSlice(configPath)

	if len(values) == 0 {
		return viper.GetString(configPath)
	}

	lines := make([]string, len(values))
	for index, value := range values {
		lines[index] = "- " + value
	}

	return strings.Join(lines, "\n")
}
