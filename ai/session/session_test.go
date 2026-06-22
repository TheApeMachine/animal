package session

import (
	"bytes"
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/ai"
	"github.com/theapemachine/animal/ai/provider"
	alcatraztool "github.com/theapemachine/animal/ai/tool/alcatraz"
	"github.com/theapemachine/qpool"
)

type fakeTerminal struct {
	readBuffer  *bytes.Buffer
	writeBuffer bytes.Buffer
}

func newFakeTerminal(output string) *fakeTerminal {
	return &fakeTerminal{readBuffer: bytes.NewBufferString(output)}
}

func (terminal *fakeTerminal) Read(payload []byte) (int, error) {
	return terminal.readBuffer.Read(payload)
}

func (terminal *fakeTerminal) Write(payload []byte) (int, error) {
	return terminal.writeBuffer.Write(payload)
}

type fakeStreamer struct {
	deltas    []string
	responses [][]string
	calls     int
	systems   []string
	contexts  [][]provider.Message
	schemas   []string
	err       error
}

func (streamer *fakeStreamer) StreamWithSink(
	system string,
	agentCtx *provider.Context,
	params *provider.Params,
	sink func(string) error,
) error {
	streamer.systems = append(streamer.systems, system)
	streamer.contexts = append(streamer.contexts, append([]provider.Message(nil), agentCtx.Messages...))

	schemaName := ""
	if params != nil && params.StructuredOutput != nil {
		schemaName = params.StructuredOutput.Name
	}

	streamer.schemas = append(streamer.schemas, schemaName)

	deltas := streamer.deltas

	if len(streamer.responses) > 0 {
		index := streamer.calls
		streamer.calls++

		if index >= len(streamer.responses) {
			index = len(streamer.responses) - 1
		}

		deltas = streamer.responses[index]
	}

	for _, delta := range deltas {
		if err := sink(delta); err != nil {
			return err
		}
	}

	return streamer.err
}

func configureSessionTestViper() {
	viper.Set("ai.prompt.template.system", "You are {{ agent.name }}, a {{ agent.role }}.")
	viper.Set("ai.prompt.template.observation", "You are {{ agent.name }}, the observation process.")
	viper.Set("ai.prompt.template.memory_recall", "You are {{ agent.name }}, the memory recall process.")
	viper.Set("ai.prompt.template.memory_consolidation", "You are {{ agent.name }}, the memory consolidation process.")
	viper.Set("project.name", "Animal")
	viper.Set("project.description", "Multi-agent coordination harness.")
}

func TestNewSession(t *testing.T) {
	Convey("Given all required session dependencies", t, func() {
		configureSessionTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		bridge, err := alcatraztool.NewBridge(ctx, newFakeTerminal("ready"))
		So(err, ShouldBeNil)

		Convey("It should create a session", func() {
			session, err := NewSession(
				ctx,
				agent,
				&fakeStreamer{deltas: []string{"pwd\n"}},
				bridge,
				provider.NewParams(),
			)

			So(err, ShouldBeNil)
			So(session, ShouldNotBeNil)
		})
	})
}

func TestCycle(t *testing.T) {
	Convey("Given a session with environment output and streamed deltas", t, func() {
		configureSessionTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		terminal := newFakeTerminal("shell ready\n")
		bridge, err := alcatraztool.NewBridge(ctx, terminal)
		So(err, ShouldBeNil)

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{deltas: []string{"make", " test", "\n"}},
			bridge,
			provider.NewParams(),
		)
		So(err, ShouldBeNil)

		Convey("It should append prompt input and stream assistant output to stdin", func() {
			result, err := session.Cycle()

			So(err, ShouldBeNil)
			So(result.Status, ShouldEqual, StatusCompleted)
			So(result.Prompt.Content, ShouldEqual, "shell ready\n")
			So(result.Assistant.Content, ShouldEqual, "make test\n")
			So(terminal.writeBuffer.String(), ShouldEqual, "make test\n")
			So(len(agent.Context.Messages), ShouldEqual, 2)
			So(agent.Context.Messages[0].Role, ShouldEqual, "user")
			So(agent.Context.Messages[1].Role, ShouldEqual, "assistant")
		})
	})
}

func TestClose(t *testing.T) {
	Convey("Given a session", t, func() {
		configureSessionTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		bridge, err := alcatraztool.NewBridge(ctx, newFakeTerminal("ready"))
		So(err, ShouldBeNil)

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{deltas: []string{"pwd\n"}},
			bridge,
			provider.NewParams(),
		)
		So(err, ShouldBeNil)

		Convey("It should cancel the session scope", func() {
			err := session.Close()

			So(err, ShouldBeNil)
			So(session.ctx.Err(), ShouldNotBeNil)
		})
	})
}

func BenchmarkCycle(benchmark *testing.B) {
	configureSessionTestViper()
	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	for benchmark.Loop() {
		agent, err := ai.NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		if err != nil {
			benchmark.Fatal(err)
		}

		bridge, err := alcatraztool.NewBridge(ctx, newFakeTerminal("ready\n"))
		if err != nil {
			benchmark.Fatal(err)
		}

		session, err := NewSession(
			ctx,
			agent,
			&fakeStreamer{deltas: []string{"pwd\n"}},
			bridge,
			provider.NewParams(),
		)
		if err != nil {
			benchmark.Fatal(err)
		}

		if _, err := session.Cycle(); err != nil {
			benchmark.Fatal(err)
		}
	}
}
