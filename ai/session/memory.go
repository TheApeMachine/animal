package session

import (
	"strings"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
)

var (
	memoryRecallStructuredOutput        = ai.MemoryRecallStructuredOutput(true)
	memoryConsolidationStructuredOutput = ai.MemoryConsolidationStructuredOutput(true)
)

func (session *Session) contextWithMemory(
	prompt provider.Message,
) (*provider.Context, ai.MemoryPacket, error) {
	if !session.agent.HasMemory() {
		return &session.agent.Context, ai.MemoryPacket{}, nil
	}

	packet, err := session.recallMemory(prompt)
	if err != nil {
		return nil, ai.MemoryPacket{}, err
	}

	agentCtx, err := session.agent.Context.Clone(session.ctx)
	if err != nil {
		return nil, ai.MemoryPacket{}, err
	}

	text := packet.Format()
	if strings.TrimSpace(text) == "" {
		return agentCtx, packet, nil
	}

	message := provider.Message{
		Role:    "system",
		Content: text,
	}

	messages := append([]provider.Message(nil), agentCtx.Messages...)
	insert := len(messages) - 1

	if insert < 0 {
		messages = append(messages, message)
		return agentCtx, packet, agentCtx.Replace(messages)
	}

	messages = append(messages[:insert], append([]provider.Message{message}, messages[insert:]...)...)

	return agentCtx, packet, agentCtx.Replace(messages)
}

func (session *Session) recallMemory(
	prompt provider.Message,
) (ai.MemoryPacket, error) {
	system, agentCtx, err := session.agent.MemoryRecallContext(
		session.ctx,
		ai.MemoryTurn{
			Prompt:   prompt,
			Messages: session.agent.Context.Messages,
		},
	)

	if err != nil {
		return ai.MemoryPacket{}, err
	}

	params := session.params.Clone().
		WithStructuredOutput(&memoryRecallStructuredOutput)
	params.WithTemperature(0)

	var builder strings.Builder

	err = session.streamer.StreamWithSink(
		system,
		agentCtx,
		params,
		func(delta string) error {
			builder.WriteString(delta)

			return nil
		},
	)

	if err != nil {
		return ai.MemoryPacket{}, err
	}

	plan, err := ai.ParseMemoryRecallPlan(builder.String())
	if err != nil {
		return ai.MemoryPacket{}, err
	}

	return session.agent.RecallMemory(session.ctx, plan)
}

func (session *Session) consolidateMemory(
	result Result,
	packet ai.MemoryPacket,
) error {
	if !session.agent.HasMemory() {
		return nil
	}

	system, agentCtx, err := session.agent.MemoryConsolidationContext(
		session.ctx,
		ai.MemoryTurn{
			Prompt:    result.Prompt,
			Assistant: result.Assistant,
			Memory:    packet,
			Messages:  session.agent.Context.Messages,
		},
	)

	if err != nil {
		return err
	}

	params := session.params.Clone().
		WithStructuredOutput(&memoryConsolidationStructuredOutput)
	params.WithTemperature(0)

	var builder strings.Builder

	err = session.streamer.StreamWithSink(
		system,
		agentCtx,
		params,
		func(delta string) error {
			builder.WriteString(delta)

			return nil
		},
	)

	if err != nil {
		return err
	}

	consolidation, err := ai.ParseMemoryConsolidation(builder.String())
	if err != nil {
		return err
	}

	return session.agent.RememberMemory(session.ctx, consolidation)
}
