.PHONY: build run clean test example-swarm-parallel-claims example-swarm-roadmap-gossip example-swarm-agent-cycle example-lease-workspace example-editor-mcp example-alcatraz-session test-alcatraz-session example-conversation-salon example-coding-horizon example-coding-horizon-dry example-memory-dmt example-memory-dmt-friction example-memory-dmt-analog example-memory-projection example-memory-manifold examples

# The pool package uses go:linkname to access runtime scheduling
# primitives (dropg, readgstatus) for zero-overhead goroutine parking.
# Go 1.26 restricts these by default; -checklinkname=0 preserves access.
LDFLAGS := -ldflags='-checklinkname=0'

# Build the animal package.
build:
	go build -o animal $(LDFLAGS)

# Run the animal package.
run:
	./animal

# Clean the animal package.
clean:
	rm -f animal

test:
	go test $(LDFLAGS) -race ./...

examples: example-swarm-parallel-claims example-swarm-roadmap-gossip example-swarm-agent-cycle example-lease-workspace example-memory-dmt example-memory-dmt-friction example-memory-dmt-analog example-memory-projection

example-swarm-parallel-claims:
	go run $(LDFLAGS) ./examples/swarm_parallel_claims

example-swarm-roadmap-gossip:
	go run $(LDFLAGS) ./examples/swarm_roadmap_gossip

example-swarm-agent-cycle:
	go run $(LDFLAGS) ./examples/swarm_agent_cycle

example-lease-workspace:
	go run $(LDFLAGS) ./examples/lease_workspace

example-editor-mcp:
	go run $(LDFLAGS) ./examples/editor_mcp

example-alcatraz-session:
	cd examples/alcatraz_session && go run $(LDFLAGS) .

test-alcatraz-session:
	cd examples/alcatraz_session && go test $(LDFLAGS) -race ./...

example-browser-mcp:
	go run $(LDFLAGS) ./examples/browser_mcp

example-conversation-salon:
	go run $(LDFLAGS) ./examples/conversation_salon

example-coding-horizon:
	go run $(LDFLAGS) ./examples/coding_horizon

example-coding-horizon-dry:
	go run $(LDFLAGS) ./examples/coding_horizon -dry-run -max-cycles 2 -workspace .

example-memory-dmt:
	go run $(LDFLAGS) ./examples/memory -mode dmt

example-memory-dmt-friction:
	go run $(LDFLAGS) ./examples/memory -mode dmt-friction

example-memory-dmt-analog:
	go run $(LDFLAGS) ./examples/memory -mode dmt-analog

example-memory-projection:
	go run $(LDFLAGS) ./examples/memory -mode projection

example-memory-manifold:
	go run $(LDFLAGS) ./examples/memory -mode manifold
