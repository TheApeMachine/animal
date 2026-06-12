# Examples

Runnable programs that exercise animal without a separate project. Run from the repository root.

All examples need the qpool linkname flag on Go 1.26+. Use the Makefile targets below.

## Swarm

| Example | Command | What it shows |
|---------|---------|---------------|
| [swarm_parallel_claims](./swarm_parallel_claims/) | `make example-swarm-parallel-claims` | Two builders claim different lane prefixes; observer sees gossip; conflicting claim is rejected |
| [swarm_roadmap_gossip](./swarm_roadmap_gossip/) | `make example-swarm-roadmap-gossip` | PM publishes `roadmap.announce`; developer merges it into a local view |
| [swarm_agent_cycle](./swarm_agent_cycle/) | `make example-swarm-agent-cycle` | Phase 1: `Agent.Cycle()` merges swarm gossip. Phase 2: calls `ai.endpoint` for an LLM reply (needs a local OpenAI-compatible server) |

## Leasing and editor

| Example | Command | What it shows |
|---------|---------|---------------|
| [lease_workspace](./lease_workspace/) | `make example-lease-workspace` | Path-prefix leases, advisory `ChangingError` on read, gated writes, FS replace |
| [editor_mcp](./editor_mcp/) | `make example-editor-mcp` | MCP SSE editor on `:3000` (set `ANIMAL_AGENT_WORKSPACE` first) |

## Conversation

| Example | Command | What it shows |
|---------|---------|---------------|
| [conversation_salon](./conversation_salon/) | `make example-conversation-salon` | Sentience panel personas, proper chat roles, persistent moderator anchor, distinctive-theme clustering. Ctrl+C to stop. |

## Coding horizon

| Example | Command | What it shows |
|---------|---------|---------------|
| [coding_horizon](./coding_horizon/) | `make example-coding-horizon-dry` | Long-horizon AI-native coding loop: observe repo, build goal+hygiene backlog, recon/plan/mutate/prove cycles with swarm announces. Dry-run skips LLM edits. |
| [coding_horizon](./coding_horizon/) | `go run -ldflags=-checklinkname=0 ./examples/coding_horizon -workspace /path/to/repo -goal "your one-line goal"` | Full loop with LLM intake, atomic replace slices, package-scoped `go test` proof, and final audit. Requires `ai.endpoint`. |

Set `-goal "..."` or pass `GOAL` via: `go run $(LDFLAGS) ./examples/coding_horizon -goal "..." -workspace /path/to/repo`.

Hygiene-only mode omits `-goal`; the loop still refactors and optimizes based on static repo analysis (oversized files, missing tests, mutex/channel signals).

## Shared helpers

[examples/support](./support/) resolves `cmd/cfg/config.yml`, builds a minimal qpool, and exposes default swarm/lease options.
