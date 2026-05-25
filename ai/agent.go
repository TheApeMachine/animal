package ai

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

type Agent struct {
	ctx           context.Context
	cancel        context.CancelFunc
	err           error
	pool          *qpool.Q
	ID            string
	Name          string
	Role          string
	System        string
	Tools         []Tool
	Memory        []Memory
	Context       provider.Context
	coopBroadcast *qpool.BroadcastGroup
	coopChannel   *qpool.Subscriber
}

func NewAgent(
	ctx context.Context,
	pool *qpool.Q,
	role, name string,
) (*Agent, error) {
	ctx, cancel := context.WithCancel(ctx)

	template := viper.GetViper().GetString("ai.prompt.template.system")

	system := strings.ReplaceAll(template, "{{ agent.role }}", role)
	system = strings.ReplaceAll(system, "{{ agent.name }}", name)
	system = strings.ReplaceAll(system, "{{ agent.characteristics }}", viper.GetViper().GetString("ai.personas."+role+".characteristics"))
	system = strings.ReplaceAll(system, "{{ agent.responsibilities }}", viper.GetViper().GetString("ai.personas."+role+".responsibilities"))
	system = strings.ReplaceAll(system, "{{ agent.guidelines }}", viper.GetViper().GetString("ai.personas."+role+".guidelines"))

	agent := &Agent{
		ctx:           ctx,
		cancel:        cancel,
		pool:          pool,
		ID:            uuid.New().String(),
		Name:          name,
		Role:          role,
		System:        strings.TrimSpace(system),
		Tools:         make([]Tool, 0),
		Memory:        make([]Memory, 0),
		Context:       *provider.NewContext(ctx),
		coopBroadcast: pool.CreateBroadcastGroup("coop", 64),
	}

	agent.coopChannel = agent.coopBroadcast.Subscribe(agent.ID, 64)

	return agent, errnie.Require(map[string]any{
		"ctx":          agent.ctx,
		"cancel":       agent.cancel,
		"pool":         agent.pool,
		"ID":           agent.ID,
		"Name":         agent.Name,
		"Role":         agent.Role,
		"System":       agent.System,
		"Tools":        agent.Tools,
		"Memory":       agent.Memory,
		"Context":      agent.Context,
	})
}

func (agent *Agent) Cycle() {
	done := false

	for !done {
		select {
		case <-agent.ctx.Done():
			return
		case qv := <-agent.coopChannel.Incoming:
			if qv == nil {
				errnie.Warn("coop channel received nil qv")
				return
			}

			raw, ok := qv.Value.(provider.Message)

			if !ok {
				errnie.Warn("coop channel received invalid qv", "qv", qv)
				return
			}

			agent.Context.Messages = append(
				agent.Context.Messages, raw,
			)
		default:
			// Do work...
		}
	}
}
