package alcatraz

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

const defaultBufferSize = 64 * 1024

/*
Bridge connects an agent conversation to an interactive Linux environment.
Environment stdout/stderr becomes user prompt input; assistant output becomes stdin.
*/
type Bridge struct {
	ctx         context.Context
	cancel      context.CancelFunc
	err         error
	terminal    io.ReadWriter
	bufferSize  int
	promptName  string
	writeSuffix string
}

/*
Option configures a Bridge.
*/
type Option func(*Bridge)

/*
WithBufferSize configures the maximum bytes read for one prompt message.
*/
func WithBufferSize(bufferSize int) Option {
	return func(bridge *Bridge) {
		bridge.bufferSize = bufferSize
	}
}

/*
WithPromptName configures the provider message name used for environment output.
*/
func WithPromptName(promptName string) Option {
	return func(bridge *Bridge) {
		bridge.promptName = promptName
	}
}

/*
WithWriteSuffix configures the bytes appended to assistant output before stdin write.
*/
func WithWriteSuffix(writeSuffix string) Option {
	return func(bridge *Bridge) {
		bridge.writeSuffix = writeSuffix
	}
}

/*
NewBridge instantiates a bridge over an io.ReadWriter such as alcatraz/pkg/environment.Session.
*/
func NewBridge(
	ctx context.Context,
	terminal io.ReadWriter,
	options ...Option,
) (*Bridge, error) {
	ctx, cancel := context.WithCancel(ctx)

	bridge := &Bridge{
		ctx:         ctx,
		cancel:      cancel,
		terminal:    terminal,
		bufferSize:  defaultBufferSize,
		promptName:  "alcatraz",
		writeSuffix: "\n",
	}

	for _, option := range options {
		option(bridge)
	}

	if bridge.bufferSize <= 0 {
		cancel()
		return nil, errnie.Err(errnie.Validation, "alcatraz bridge buffer size is required", nil)
	}

	return bridge, errnie.Require(map[string]any{
		"ctx":        bridge.ctx,
		"cancel":     bridge.cancel,
		"terminal":   bridge.terminal,
		"bufferSize": bridge.bufferSize,
	})
}

/*
ReadPrompt reads one stdout/stderr chunk and returns it as a user prompt message.
*/
func (bridge *Bridge) ReadPrompt() (provider.Message, error) {
	return bridge.ReadPromptN(bridge.bufferSize)
}

/*
ReadPromptN reads up to maxBytes from stdout/stderr into a provider message.
*/
func (bridge *Bridge) ReadPromptN(maxBytes int) (provider.Message, error) {
	if maxBytes <= 0 {
		return provider.Message{}, errnie.Err(
			errnie.Validation,
			"alcatraz bridge max bytes is required",
			nil,
		)
	}

	payload := make([]byte, maxBytes)
	count, err := bridge.terminal.Read(payload)

	if count == 0 && err != nil {
		return provider.Message{}, err
	}

	if count == 0 {
		return provider.Message{}, errnie.Err(
			errnie.IO,
			"alcatraz bridge read returned no output",
			nil,
		)
	}

	message := provider.Message{
		Name:    bridge.promptName,
		Role:    "user",
		Content: string(payload[:count]),
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return message, err
	}

	return message, nil
}

/*
AppendPrompt reads environment output and appends it to the agent context.
*/
func (bridge *Bridge) AppendPrompt(agentCtx *provider.Context) error {
	message, err := bridge.ReadPrompt()
	if err != nil {
		return err
	}

	return agentCtx.Append(message)
}

/*
WriteMessage writes assistant output to environment stdin.
*/
func (bridge *Bridge) WriteMessage(message provider.Message) error {
	role := strings.ToLower(strings.TrimSpace(message.Role))

	if role != "" && role != "assistant" {
		return errnie.Err(
			errnie.Validation,
			"alcatraz bridge writes assistant messages only",
			nil,
		)
	}

	_, err := bridge.writeContent(message.Content)

	return err
}

/*
WriteOutput writes assistant text to environment stdin.
*/
func (bridge *Bridge) WriteOutput(content string) error {
	_, err := bridge.writeContent(content)

	return err
}

/*
WriteChunk writes one streamed assistant delta to environment stdin unchanged.
*/
func (bridge *Bridge) WriteChunk(content string) error {
	if content == "" {
		return errnie.Err(
			errnie.Validation,
			"alcatraz bridge output chunk is required",
			nil,
		)
	}

	_, err := bridge.terminal.Write([]byte(content))

	return err
}

/*
WriteArtifact writes a streamed text artifact to environment stdin.
*/
func (bridge *Bridge) WriteArtifact(artifact *datura.Artifact) error {
	content, err := qpool.ArtifactValue[string](artifact)
	if err != nil {
		return err
	}

	return bridge.WriteOutput(content)
}

func (bridge *Bridge) writeContent(content string) (int, error) {
	if strings.TrimSpace(content) == "" {
		return 0, errnie.Err(
			errnie.Validation,
			"alcatraz bridge output content is required",
			nil,
		)
	}

	return bridge.terminal.Write([]byte(content + bridge.writeSuffix))
}

/*
Read exposes the environment stdout/stderr stream.
*/
func (bridge *Bridge) Read(payload []byte) (int, error) {
	return bridge.terminal.Read(payload)
}

/*
Write exposes the environment stdin stream.
*/
func (bridge *Bridge) Write(payload []byte) (int, error) {
	return bridge.terminal.Write(payload)
}

/*
Close cancels the bridge without owning the underlying environment lifecycle.
*/
func (bridge *Bridge) Close() error {
	bridge.cancel()

	return nil
}
