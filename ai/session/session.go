package session

import (
	"context"
	"strings"
	"time"

	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	alcatraztool "github.com/theapemachine/animal/ai/tool/alcatraz"
	"github.com/theapemachine/errnie"
)

/*
Streamer is the provider surface required for interactive sessions.
*/
type Streamer interface {
	StreamWithSink(
		system string,
		agentCtx *provider.Context,
		params *provider.Params,
		sink func(string) error,
	) error
}

/*
Status identifies the terminal state of one session cycle.
*/
type Status string

const (
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

/*
Result captures one prompt-to-stdin cycle.
*/
type Result struct {
	Status    Status
	Prompt    provider.Message
	Assistant provider.Message
	StartedAt time.Time
	EndedAt   time.Time
}

/*
Session binds an agent, streamer, and Linux stdio bridge.
*/
type Session struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	agent    *ai.Agent
	streamer Streamer
	bridge   *alcatraztool.Bridge
	params   *provider.Params
}

/*
NewSession instantiates an interactive agent session.
*/
func NewSession(
	ctx context.Context,
	agent *ai.Agent,
	streamer Streamer,
	bridge *alcatraztool.Bridge,
	params *provider.Params,
) (*Session, error) {
	ctx, cancel := context.WithCancel(ctx)

	session := &Session{
		ctx:      ctx,
		cancel:   cancel,
		agent:    agent,
		streamer: streamer,
		bridge:   bridge,
		params:   params,
	}

	return session, errnie.Require(map[string]any{
		"ctx":      session.ctx,
		"cancel":   session.cancel,
		"agent":    session.agent,
		"streamer": session.streamer,
		"bridge":   session.bridge,
		"params":   session.params,
	})
}

/*
Cycle sends environment output to the agent and streams assistant output to stdin.
*/
func (session *Session) Cycle() (Result, error) {
	result := Result{
		StartedAt: time.Now().UTC(),
		Status:    StatusFailed,
	}

	prompt, err := session.bridge.ReadPrompt()
	if err != nil {
		result.EndedAt = time.Now().UTC()
		return result, err
	}

	result.Prompt = prompt

	if err := session.agent.Context.Append(prompt); err != nil {
		result.EndedAt = time.Now().UTC()
		return result, err
	}

	assistant, err := session.streamAssistant()
	result.Assistant = assistant
	result.EndedAt = time.Now().UTC()

	if err != nil {
		return result, err
	}

	if err := session.agent.Context.Append(assistant); err != nil {
		return result, err
	}

	result.Status = StatusCompleted

	return result, nil
}

func (session *Session) streamAssistant() (provider.Message, error) {
	var builder strings.Builder

	err := session.streamer.StreamWithSink(
		session.agent.System,
		&session.agent.Context,
		session.params,
		func(delta string) error {
			builder.WriteString(delta)

			return session.bridge.WriteChunk(delta)
		},
	)

	if err != nil {
		return provider.Message{Role: "assistant", Content: builder.String()}, err
	}

	if strings.TrimSpace(builder.String()) == "" {
		return provider.Message{}, errnie.Err(
			errnie.Validation,
			"session assistant output is required",
			nil,
		)
	}

	return provider.Message{
		Role:    "assistant",
		Content: builder.String(),
	}, nil
}

/*
Close cancels the session scope.
*/
func (session *Session) Close() error {
	session.cancel()

	return nil
}
