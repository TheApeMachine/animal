package alcatraz

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/qpool"
)

type scriptTerminal struct {
	readBuffer  *bytes.Buffer
	writeBuffer bytes.Buffer
}

func newScriptTerminal(output string) *scriptTerminal {
	return &scriptTerminal{
		readBuffer: bytes.NewBufferString(output),
	}
}

func (terminal *scriptTerminal) Read(payload []byte) (int, error) {
	return terminal.readBuffer.Read(payload)
}

func (terminal *scriptTerminal) Write(payload []byte) (int, error) {
	return terminal.writeBuffer.Write(payload)
}

func TestNewBridge(t *testing.T) {
	Convey("Given an interactive terminal", t, func() {
		terminal := newScriptTerminal("ready\n")

		Convey("When NewBridge is called", func() {
			bridge, err := NewBridge(context.Background(), terminal, WithBufferSize(32))

			Convey("Then it should create a bridge", func() {
				So(err, ShouldBeNil)
				So(bridge, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an invalid buffer size", t, func() {
		terminal := newScriptTerminal("ready\n")

		Convey("When NewBridge is called", func() {
			bridge, err := NewBridge(context.Background(), terminal, WithBufferSize(0))

			Convey("Then it should reject the bridge", func() {
				So(bridge, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestReadPrompt(t *testing.T) {
	Convey("Given environment stdout/stderr", t, func() {
		bridge, err := NewBridge(
			context.Background(),
			newScriptTerminal("make test\nPASS\n"),
			WithPromptName("linux"),
		)
		So(err, ShouldBeNil)

		Convey("When ReadPrompt is called", func() {
			message, readErr := bridge.ReadPrompt()

			Convey("Then stdout/stderr should become a user prompt message", func() {
				So(readErr, ShouldBeNil)
				So(message.Role, ShouldEqual, "user")
				So(message.Name, ShouldEqual, "linux")
				So(message.Content, ShouldEqual, "make test\nPASS\n")
			})
		})
	})
}

func TestReadPromptN(t *testing.T) {
	Convey("Given a small read limit", t, func() {
		bridge, err := NewBridge(context.Background(), newScriptTerminal("abcdef"))
		So(err, ShouldBeNil)

		Convey("When ReadPromptN is called", func() {
			message, readErr := bridge.ReadPromptN(3)

			Convey("Then it should read at most the requested bytes", func() {
				So(readErr, ShouldBeNil)
				So(message.Content, ShouldEqual, "abc")
			})
		})
	})
}

func TestAppendPrompt(t *testing.T) {
	Convey("Given a provider context and environment output", t, func() {
		bridge, err := NewBridge(context.Background(), newScriptTerminal("diagnostic\n"))
		So(err, ShouldBeNil)
		agentCtx := provider.NewContext(context.Background())

		Convey("When AppendPrompt is called", func() {
			appendErr := bridge.AppendPrompt(agentCtx)

			Convey("Then it should append a user message", func() {
				So(appendErr, ShouldBeNil)
				So(len(agentCtx.Messages), ShouldEqual, 1)
				So(agentCtx.Messages[0].Content, ShouldEqual, "diagnostic\n")
			})
		})
	})
}

func TestWriteMessage(t *testing.T) {
	Convey("Given an assistant message", t, func() {
		terminal := newScriptTerminal("")
		bridge, err := NewBridge(context.Background(), terminal)
		So(err, ShouldBeNil)

		Convey("When WriteMessage is called", func() {
			writeErr := bridge.WriteMessage(provider.Message{
				Role:    "assistant",
				Content: "go test ./...",
			})

			Convey("Then assistant output should be written to stdin", func() {
				So(writeErr, ShouldBeNil)
				So(terminal.writeBuffer.String(), ShouldEqual, "go test ./...\n")
			})
		})
	})
}

func TestWriteOutput(t *testing.T) {
	Convey("Given an output string", t, func() {
		terminal := newScriptTerminal("")
		bridge, err := NewBridge(context.Background(), terminal, WithWriteSuffix(""))
		So(err, ShouldBeNil)

		Convey("When WriteOutput is called", func() {
			writeErr := bridge.WriteOutput("pwd")

			Convey("Then it should write the exact output to stdin", func() {
				So(writeErr, ShouldBeNil)
				So(terminal.writeBuffer.String(), ShouldEqual, "pwd")
			})
		})
	})
}

func TestWriteChunk(t *testing.T) {
	Convey("Given a streamed output chunk", t, func() {
		terminal := newScriptTerminal("")
		bridge, err := NewBridge(context.Background(), terminal)
		So(err, ShouldBeNil)

		Convey("When WriteChunk is called", func() {
			writeErr := bridge.WriteChunk("go ")

			Convey("Then it should write the chunk without suffix", func() {
				So(writeErr, ShouldBeNil)
				So(terminal.writeBuffer.String(), ShouldEqual, "go ")
			})
		})
	})
}

func TestWriteArtifact(t *testing.T) {
	Convey("Given a streamed text artifact", t, func() {
		terminal := newScriptTerminal("")
		bridge, err := NewBridge(context.Background(), terminal)
		So(err, ShouldBeNil)

		artifact, artifactErr := qpool.NewBusArtifact(
			"agent",
			"alcatraz",
			"text",
			"ls",
			time.Minute,
		)
		So(artifactErr, ShouldBeNil)

		Convey("When WriteArtifact is called", func() {
			writeErr := bridge.WriteArtifact(artifact)

			Convey("Then the artifact text should be written to stdin", func() {
				So(writeErr, ShouldBeNil)
				So(terminal.writeBuffer.String(), ShouldEqual, "ls\n")
			})
		})
	})
}

func TestRead(t *testing.T) {
	Convey("Given a bridge", t, func() {
		bridge, err := NewBridge(context.Background(), newScriptTerminal("abc"))
		So(err, ShouldBeNil)

		Convey("When Read is called directly", func() {
			payload := make([]byte, 2)
			count, readErr := bridge.Read(payload)

			Convey("Then it should delegate to the terminal", func() {
				So(readErr, ShouldBeNil)
				So(count, ShouldEqual, 2)
				So(string(payload), ShouldEqual, "ab")
			})
		})
	})
}

func TestWrite(t *testing.T) {
	Convey("Given a bridge", t, func() {
		terminal := newScriptTerminal("")
		bridge, err := NewBridge(context.Background(), terminal)
		So(err, ShouldBeNil)

		Convey("When Write is called directly", func() {
			count, writeErr := bridge.Write([]byte("x"))

			Convey("Then it should delegate to the terminal", func() {
				So(writeErr, ShouldBeNil)
				So(count, ShouldEqual, 1)
				So(terminal.writeBuffer.String(), ShouldEqual, "x")
			})
		})
	})
}

func TestClose(t *testing.T) {
	Convey("Given a bridge", t, func() {
		bridge, err := NewBridge(context.Background(), newScriptTerminal(""))
		So(err, ShouldBeNil)

		Convey("When Close is called", func() {
			closeErr := bridge.Close()

			Convey("Then it should cancel without closing the terminal", func() {
				So(closeErr, ShouldBeNil)
				So(bridge.ctx.Err(), ShouldNotBeNil)
			})
		})
	})
}

func BenchmarkReadPrompt(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 1024)

	for b.Loop() {
		bridge, err := NewBridge(context.Background(), newScriptTerminal(string(payload)))
		if err != nil {
			b.Fatal(err)
		}

		if _, err := bridge.ReadPrompt(); err != nil && err != io.EOF {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteOutput(b *testing.B) {
	terminal := newScriptTerminal("")
	bridge, err := NewBridge(context.Background(), terminal)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		if err := bridge.WriteOutput("go test ./..."); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteChunk(b *testing.B) {
	terminal := newScriptTerminal("")
	bridge, err := NewBridge(context.Background(), terminal)
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		if err := bridge.WriteChunk("x"); err != nil {
			b.Fatal(err)
		}
	}
}
