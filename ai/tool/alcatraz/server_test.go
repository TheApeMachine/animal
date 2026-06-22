package alcatraz

import (
	"context"
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/ai/mcpclient"
)

func TestMCPServer(t *testing.T) {
	Convey("Given an alcatraz bridge", t, func() {
		ctx := context.Background()
		bridge, err := NewBridge(ctx, newScriptTerminal("linux output"))
		So(err, ShouldBeNil)

		server, err := NewServer(ctx, bridge)
		So(err, ShouldBeNil)

		Convey("It should return an MCP server", func() {
			mcpServer := server.MCPServer()

			So(mcpServer, ShouldNotBeNil)
		})
	})
}

func TestNewServer(t *testing.T) {
	Convey("Given an alcatraz bridge", t, func() {
		ctx := context.Background()
		bridge, err := NewBridge(ctx, newScriptTerminal(""))
		So(err, ShouldBeNil)

		Convey("It should create a server adapter", func() {
			server, err := NewServer(ctx, bridge)

			So(err, ShouldBeNil)
			So(server, ShouldNotBeNil)
		})
	})
}

func TestReadTool(t *testing.T) {
	Convey("Given an MCP client connected to the bridge", t, func() {
		ctx := context.Background()
		bridge, err := NewBridge(ctx, newScriptTerminal("stdout\nstderr\n"))
		So(err, ShouldBeNil)

		server, err := NewServer(ctx, bridge)
		So(err, ShouldBeNil)

		session, err := mcpclient.ConnectInMemory(ctx, server.MCPServer())
		So(err, ShouldBeNil)

		Convey("It should return environment output", func() {
			payload, err := mcpclient.CallToolJSON(
				ctx,
				session,
				"alcatraz_read",
				ReadParams{MaxBytes: 64},
			)
			var result ReadResult
			decodeErr := json.Unmarshal(payload, &result)

			So(err, ShouldBeNil)
			So(decodeErr, ShouldBeNil)
			So(result.Content, ShouldEqual, "stdout\nstderr\n")
			So(result.Bytes, ShouldEqual, len("stdout\nstderr\n"))
		})
	})
}

func TestWriteTool(t *testing.T) {
	Convey("Given an MCP client connected to the bridge", t, func() {
		ctx := context.Background()
		terminal := newScriptTerminal("")
		bridge, err := NewBridge(ctx, terminal)
		So(err, ShouldBeNil)

		server, err := NewServer(ctx, bridge)
		So(err, ShouldBeNil)

		session, err := mcpclient.ConnectInMemory(ctx, server.MCPServer())
		So(err, ShouldBeNil)

		Convey("It should write to environment stdin", func() {
			payload, err := mcpclient.CallToolJSON(
				ctx,
				session,
				"alcatraz_write",
				WriteParams{Content: "make test"},
			)
			var result WriteResult
			decodeErr := json.Unmarshal(payload, &result)

			So(err, ShouldBeNil)
			So(decodeErr, ShouldBeNil)
			So(result.Bytes, ShouldEqual, len("make test\n"))
			So(terminal.writeBuffer.String(), ShouldEqual, "make test\n")
		})
	})
}

func BenchmarkWriteTool(benchmark *testing.B) {
	ctx := context.Background()
	terminal := newScriptTerminal("")
	bridge, err := NewBridge(ctx, terminal)
	if err != nil {
		benchmark.Fatal(err)
	}

	server, err := NewServer(ctx, bridge)
	if err != nil {
		benchmark.Fatal(err)
	}

	session, err := mcpclient.ConnectInMemory(ctx, server.MCPServer())
	if err != nil {
		benchmark.Fatal(err)
	}

	for benchmark.Loop() {
		if _, err := mcpclient.CallToolJSON(
			ctx,
			session,
			"alcatraz_write",
			WriteParams{Content: "pwd"},
		); err != nil {
			benchmark.Fatal(err)
		}
	}
}
