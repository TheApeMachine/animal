package main

import (
	"context"
	"fmt"
	"os"
	osexec "os/exec"
	"strings"
	"time"

	"github.com/spf13/viper"
	alcatrazconfig "github.com/theapemachine/alcatraz/pkg/config"
	alcatrazenv "github.com/theapemachine/alcatraz/pkg/environment"
	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	animalsession "github.com/theapemachine/animal/ai/session"
	alcatraztool "github.com/theapemachine/animal/ai/tool/alcatraz"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

const startupScript = "printf 'alcatraz ready\\n'; IFS= read -r command; /bin/sh -c \"$command\""

var dockerInfo = func(ctx context.Context) ([]byte, error) {
	return osexec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}").CombinedOutput()
}

/*
ScriptedStreamer streams deterministic assistant output into a session sink.
It lets the example prove the alcatraz stdio bridge without requiring an LLM endpoint.
*/
type ScriptedStreamer struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	chunks []string
}

/*
NewScriptedStreamer instantiates a deterministic streamer for example runs.
*/
func NewScriptedStreamer(
	ctx context.Context,
	chunks ...string,
) (*ScriptedStreamer, error) {
	ctx, cancel := context.WithCancel(ctx)

	if len(chunks) == 0 {
		cancel()

		return nil, errnie.Error(
			errnie.Err(errnie.Validation, "scripted streamer chunks are required", nil),
		)
	}

	streamer := &ScriptedStreamer{
		ctx:    ctx,
		cancel: cancel,
		chunks: append([]string(nil), chunks...),
	}

	return streamer, errnie.Require(map[string]any{
		"ctx":    streamer.ctx,
		"cancel": streamer.cancel,
		"chunks": streamer.chunks,
	})
}

/*
StreamWithSink writes the configured chunks to the streaming sink.
*/
func (scriptedStreamer *ScriptedStreamer) StreamWithSink(
	system string,
	agentCtx *provider.Context,
	params *provider.Params,
	sink func(string) error,
) error {
	if len(scriptedStreamer.chunks) == 0 {
		return errnie.Error(
			errnie.Err(errnie.Validation, "scripted streamer chunks are required", nil),
		)
	}

	if err := errnie.Require(map[string]any{"sink": sink}); err != nil {
		return err
	}

	for _, chunk := range scriptedStreamer.chunks {
		if err := sink(chunk); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "alcatraz session: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if err := requireDocker(ctx); err != nil {
		return err
	}

	configureAnimal()

	agentEnvironment, err := startEnvironment(ctx)
	if err != nil {
		return err
	}
	defer agentEnvironment.Close()

	terminal, err := agentEnvironment.Attach("/bin/sh", "-c", startupScript)
	if err != nil {
		return err
	}
	defer terminal.Close()

	bridge, err := alcatraztool.NewBridge(ctx, terminal, alcatraztool.WithBufferSize(4096))
	if err != nil {
		return err
	}
	defer bridge.Close()

	agent, err := newAgent(ctx)
	if err != nil {
		return err
	}

	streamer, err := NewScriptedStreamer(ctx, "printf ", "'cycle-ok\\n'", "\n")
	if err != nil {
		return err
	}

	session, err := animalsession.NewSession(ctx, agent, streamer, bridge, provider.NewParams())
	if err != nil {
		return err
	}
	defer session.Close()

	return runSessionCycle(session, bridge)
}

func runSessionCycle(
	session *animalsession.Session,
	bridge *alcatraztool.Bridge,
) error {
	result, err := session.Cycle()
	if err != nil {
		return err
	}

	response, err := bridge.ReadPromptN(4096)
	if err != nil {
		return err
	}

	fmt.Printf("prompt: %s", result.Prompt.Content)
	fmt.Printf("assistant: %s", result.Assistant.Content)
	fmt.Printf("response: %s", response.Content)

	return nil
}

func requireDocker(ctx context.Context) error {
	output, err := dockerInfo(ctx)

	if err == nil {
		return nil
	}

	return errnie.Error(errnie.Err(
		errnie.IO,
		"docker daemon is required for alcatraz session example: "+strings.TrimSpace(string(output)),
		err,
	))
}

func configureAnimal() {
	viper.Set("ai.prompt.template.system", "You are {{ agent.name }}, a {{ agent.role }}.")
	viper.Set("project.name", "Animal")
	viper.Set("project.description", "Alcatraz session proof.")
}

func startEnvironment(ctx context.Context) (*alcatrazenv.AgentEnvironment, error) {
	identifier := fmt.Sprintf("animal-session-%d", time.Now().UnixNano())
	globalConfig := alcatrazconfig.NewEnvironmentConfig()
	agentEnvironment := alcatrazenv.NewAgentEnvironment(
		ctx,
		alcatrazenv.AgentEnvironmentConfigFromGlobal(identifier, identifier, globalConfig),
	)

	if err := agentEnvironment.Start(); err != nil {
		return nil, err
	}

	return agentEnvironment, nil
}

func newAgent(ctx context.Context) (*ai.Agent, error) {
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	return ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
}
