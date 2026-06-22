# Examples

Runnable programs that exercise animal without a separate project. Run from the repository root.

All examples need the qpool linkname flag on Go 1.26+. Use the Makefile targets below.

## Swarm

| Example                                           | Command                              | What it shows                                                                                                                        |
|---------------------------------------------------|--------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------|
| [swarm_parallel_claims](./swarm_parallel_claims/) | `make example-swarm-parallel-claims` | Two builders claim different lane prefixes; observer sees gossip; conflicting claim is rejected                                      |
| [swarm_roadmap_gossip](./swarm_roadmap_gossip/)   | `make example-swarm-roadmap-gossip`  | PM publishes `roadmap.announce`; developer merges it into a local view                                                               |
| [swarm_agent_cycle](./swarm_agent_cycle/)         | `make example-swarm-agent-cycle`     | Phase 1: `Agent.Cycle()` merges swarm gossip. Phase 2: calls `ai.endpoint` for an LLM reply (needs a local OpenAI-compatible server) |

The swarm package also exposes generic A2A task broadcasts, task lifecycle events, submitted-task queries, blocker queries, friction/quality/opportunity signals, and normalized success metrics. These use the same mesh and local `View` merge path as gossip and leases.

## Leasing and editor

| Example                               | Command                        | What it shows                                                                  |
|---------------------------------------|--------------------------------|--------------------------------------------------------------------------------|
| [lease_workspace](./lease_workspace/) | `make example-lease-workspace` | Path-prefix leases, advisory `ChangingError` on read, gated writes, FS replace |
| [editor_mcp](./editor_mcp/)           | `make example-editor-mcp`      | MCP SSE editor on `:3000` (set `ANIMAL_AGENT_WORKSPACE` first)                 |

`ai/tool/alcatraz` can wrap an `io.ReadWriter` such as `github.com/theapemachine/alcatraz/pkg/environment.Session`: environment stdout/stderr is read as agent prompt input, and assistant output is written to environment stdin. It also exposes `alcatraz_read` and `alcatraz_write` MCP tools.

`ai/session` binds an `ai.Agent`, streaming provider, and alcatraz bridge so streamed model deltas are written to stdin as they arrive. It can also run an A2A task by cloning the agent, claiming a `lease_prefix` metadata value when present, and reporting completion metrics or blocker signals through swarm.

## Alcatraz session

| Example                                 | Command                         | What it shows                                                                                                                            |
|-----------------------------------------|---------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| [alcatraz_session](./alcatraz_session/) | `make example-alcatraz-session` | Starts a hardened alcatraz environment, attaches a live exec stream, wraps it with `ai/tool/alcatraz`, and runs one `ai/session.Cycle()` |

Run `make test-alcatraz-session` for the nested example module tests. Docker is required for the runnable example.

## Conversation

| Example                                     | Command                           | What it shows                                                                                                           |
|---------------------------------------------|-----------------------------------|-------------------------------------------------------------------------------------------------------------------------|
| [conversation_salon](./conversation_salon/) | `make example-conversation-salon` | Sentience panel personas, proper chat roles, persistent moderator anchor, distinctive-theme clustering. Ctrl+C to stop. |

## Coding horizon

| Example                             | Command                                                                                                          | What it shows                                                                                                                                               |
|-------------------------------------|------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [coding_horizon](./coding_horizon/) | `make example-coding-horizon-dry`                                                                                | Long-horizon AI-native coding loop: observe repo, build goal+hygiene backlog, recon/plan/mutate/prove cycles with swarm announces. Dry-run skips LLM edits. |
| [coding_horizon](./coding_horizon/) | `go run -ldflags=-checklinkname=0 ./examples/coding_horizon -workspace /path/to/repo -goal "your one-line goal"` | Full loop with LLM intake, atomic replace slices, package-scoped `go test` proof, and final audit. Requires `ai.endpoint`.                                  |

Set `-goal "..."` or pass `GOAL` via: `go run $(LDFLAGS) ./examples/coding_horizon -goal "..." -workspace /path/to/repo`.

Hygiene-only mode omits `-goal`; the loop still refactors and optimizes based on static repo analysis (oversized files, missing tests, mutex/channel signals).

## Memory

| Example             | Command                        | What it shows                                                                                                                        |
|---------------------|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------------|
| [memory](./memory/) | `make example-memory-dmt`      | DMT cognitive memory: durable records enter the radix forest, train episodic/REM sensory paths, and recall documents plus relations. |
| [memory](./memory/) | `make example-memory-dmt-friction` | Swarm hygiene backlog: friction/quality observations are remembered, related, and recalled for agents looking for useful work.      |
| [memory](./memory/) | `make example-memory-dmt-analog` | Structural analog recall: a new-but-similar lease conflict recalls the closest prior task route through DMT shared-prefix search.    |
| [memory](./memory/) | `make example-memory-projection` | Projection-aware memory: a local projector enriches records with embeddings before DMT storage.                                      |
| [memory](./memory/) | `make example-memory-manifold` | Optional nomagique manifold projection: latent embeddings, energy, and surprise are produced before memory consolidation. Requires darwin+cgo/Metal. |

## Shared helpers

[examples/support](./support/) resolves `cmd/cfg/config.yml`, builds a minimal qpool, and exposes default swarm/lease options.
