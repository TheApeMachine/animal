package ai

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/storage"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/datura"
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
	registry      *swarm.Registry
	claimPrefixes []string
	training      TrainingSink
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
		registry:      registry,
		claimPrefixes: append([]string(nil), claimPrefixes...),
	}

	if registry != nil {
		participant, err := registry.NewParticipant(agent.ID, name, role, claimPrefixes)

		if err != nil {
			return nil, err
		}

		agent.participant = participant
	}

	if registry == nil {
		agent.coopBroadcast = pool.CreateBroadcastGroup("coop")
		agent.coopChannel = agent.coopBroadcast.Acquire(agent.ID, nil)
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

/*
Clone creates a new agent from the current context and appends one sub-task message.
*/
func (agent *Agent) Clone(ctx context.Context, subTask string) (*Agent, error) {
	subTask = strings.TrimSpace(subTask)

	if subTask == "" {
		return nil, errnie.Err(errnie.Validation, "agent clone sub-task is required", nil)
	}

	return agent.CloneWithMessage(ctx, provider.Message{
		Role:    "user",
		Content: subTask,
	})
}

/*
CloneWithTask creates a clone from an A2A task instruction.
*/
func (agent *Agent) CloneWithTask(ctx context.Context, task a2a.Task) (*Agent, error) {
	if err := task.Validate(); err != nil {
		return nil, err
	}

	instruction := strings.TrimSpace(task.Instruction())

	if instruction == "" {
		return nil, errnie.Err(errnie.Validation, "agent clone task instruction is required", nil)
	}

	return agent.Clone(ctx, instruction)
}

/*
CloneWithMessage creates a clone and appends one provider message.
*/
func (agent *Agent) CloneWithMessage(
	ctx context.Context,
	message provider.Message,
) (*Agent, error) {
	ctx, cancel := context.WithCancel(ctx)

	agentCtx, err := agent.Context.Clone(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	if err := agentCtx.Append(message); err != nil {
		cancel()
		return nil, err
	}

	clone := &Agent{
		ctx:           ctx,
		cancel:        cancel,
		pool:          agent.pool,
		ID:            uuid.New().String(),
		Name:          agent.Name,
		Role:          agent.Role,
		System:        agent.System,
		Tools:         append(make([]Tool, 0, len(agent.Tools)), agent.Tools...),
		Memory:        append(make([]Memory, 0, len(agent.Memory)), agent.Memory...),
		Context:       *agentCtx,
		registry:      agent.registry,
		claimPrefixes: append([]string(nil), agent.claimPrefixes...),
		training:      agent.training,
	}

	if clone.registry != nil {
		participant, err := clone.registry.NewParticipant(
			clone.ID,
			clone.Name,
			clone.Role,
			clone.claimPrefixes,
		)

		if err != nil {
			clone.cancel()
			return nil, err
		}

		clone.participant = participant
	}

	if clone.registry == nil {
		clone.coopBroadcast = clone.pool.CreateBroadcastGroup("coop")
		clone.coopChannel = clone.coopBroadcast.Acquire(clone.ID, nil)
	}

	return clone, errnie.Require(map[string]any{
		"ctx":     clone.ctx,
		"cancel":  clone.cancel,
		"pool":    clone.pool,
		"ID":      clone.ID,
		"Name":    clone.Name,
		"Role":    clone.Role,
		"System":  clone.System,
		"Tools":   clone.Tools,
		"Memory":  clone.Memory,
		"Context": clone.Context,
	})
}

/*
SwapContext hot-swaps an agent onto a different message history.
*/
func (agent *Agent) SwapContext(agentCtx *provider.Context) error {
	clone, err := agentCtx.Clone(agent.ctx)

	if err != nil {
		return err
	}

	agent.Context = *clone

	return nil
}

/*
UseTrainingRecorder attaches automatic fine-tuning trace collection to the agent.
*/
func (agent *Agent) UseTrainingRecorder(trainingRecorder *TrainingRecorder) error {
	if trainingRecorder == nil {
		return errnie.Err(errnie.Validation, "training recorder is required", nil)
	}

	return agent.UseTrainingSink(trainingRecorder)
}

/*
UseTrainingStore opens configured artifact-backed training capture and attaches it to the agent.
*/
func (agent *Agent) UseTrainingStore(
	ctx context.Context,
	config storage.Config,
) (*TrainingStore, error) {
	trainingStore, err := NewTrainingStoreFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := agent.UseTrainingSink(trainingStore); err != nil {
		return nil, err
	}

	return trainingStore, nil
}

/*
UseTrainingSink attaches automatic fine-tuning trace collection to the agent.
*/
func (agent *Agent) UseTrainingSink(trainingSink TrainingSink) error {
	if trainingSink == nil {
		return errnie.Err(errnie.Validation, "training sink is required", nil)
	}

	agent.training = trainingSink

	return nil
}

func (agent *Agent) Cycle() {
	for {
		select {
		case <-agent.ctx.Done():
			return
		default:
		}

		artifact := agent.pollIncoming()
		if artifact != nil {
			agent.handleIncoming(artifact)
			continue
		}

		if agent.participant != nil {
			if err := agent.participant.Drain(); err != nil {
				errnie.Warn("swarm drain failed", "err", err)
				return
			}
		}

		return
	}
}

func (agent *Agent) pollIncoming() *datura.Artifact {
	if agent.participant != nil {
		return agent.participant.Poll()
	}

	if agent.coopChannel == nil {
		return nil
	}

	return agent.coopChannel.Poll()
}

func (agent *Agent) handleIncoming(artifact *datura.Artifact) {
	if artifact == nil {
		return
	}

	switch qpool.BusMessageType(artifact) {
	case swarm.MessageTypeRumor,
		swarm.MessageTypeTask,
		swarm.MessageTypeTaskStatus,
		swarm.MessageTypeSignal,
		swarm.MessageTypeMetric:
		if agent.participant == nil {
			errnie.Warn("agent received swarm artifact without participant")
			return
		}

		if err := agent.participant.ReceiveArtifact(artifact); err != nil {
			errnie.Warn("swarm receive failed", "err", err)
			return
		}

		if qpool.BusMessageType(artifact) == swarm.MessageTypeMetric {
			agent.recordMetric(artifact)
		}
	default:
		payload := datura.As[provider.Message](artifact)

		if payload.Role == "" && payload.Content == "" {
			errnie.Warn("agent channel received invalid payload type", "type", qpool.BusMessageType(artifact))
			return
		}

		agent.Context.Messages = append(agent.Context.Messages, payload)
	}
}

func (agent *Agent) recordMetric(artifact *datura.Artifact) {
	if agent.training == nil {
		return
	}

	metric := datura.As[swarm.Metric](artifact)

	if metric.ActorID != agent.ID || !metric.Success {
		return
	}

	if err := agent.training.Record(agent, metric); err != nil {
		errnie.Warn("agent training record failed", "err", err)
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
