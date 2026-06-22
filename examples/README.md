# Examples

Runnable programs that exercise `animal` end-to-end — no separate project required. Each one is a real, self-contained `main` you can read, run, and adapt.

**Run every command from the repository root.** All examples need the `qpool` linkname flag on Go 1.26+; the `make` targets below set `-ldflags='-checklinkname=0'` for you, so prefer them.

> **Need a model?** Examples marked 🧠 call an OpenAI-compatible endpoint (`ai.endpoint` in `cmd/cfg/config.yml`, default `http://localhost:1234/v1`). Start a local server (e.g. [LM Studio](https://lmstudio.ai) or [vLLM](https://github.com/vllm-project/vllm)) and set `OPENAI_API_KEY` if it requires one. Examples without 🧠 run offline.

---

## Where to start

| If you want to...                              | Run this                            |
|------------------------------------------------|-------------------------------------|
| See agents coordinate with **no model needed** | `make example-swarm-roadmap-gossip` |
| See a full **agent absorb gossip and reply**   | `make example-swarm-agent-cycle` 🧠 |
| Understand **memory**                          | `make example-memory-dmt`           |
| Watch a **long-horizon coding loop**           | `make example-coding-horizon-dry`   |

---

## Swarm — coordination without a boss

Agents share state, claim work, and broadcast progress over a gossip mesh. No agent is in charge.

| Example                                           | Command                              | What it shows                                                                                                                   |
|---------------------------------------------------|--------------------------------------|---------------------------------------------------------------------------------------------------------------------------------|
| [swarm_parallel_claims](./swarm_parallel_claims/) | `make example-swarm-parallel-claims` | Two builders claim different lane prefixes; an observer sees the gossip; a conflicting claim is rejected.                       |
| [swarm_roadmap_gossip](./swarm_roadmap_gossip/)   | `make example-swarm-roadmap-gossip`  | A PM publishes `roadmap.announce`; a developer merges it into a local view.                                                     |
| [swarm_agent_cycle](./swarm_agent_cycle/)         | `make example-swarm-agent-cycle` 🧠  | **Phase 1:** `Agent.Cycle()` merges swarm gossip into context. **Phase 2:** streams an LLM reply that acknowledges the roadmap. |

Beyond gossip and leases, the `swarm` package exposes generic A2A task broadcasts, task lifecycle events, submitted-task and blocker queries, friction/quality/opportunity signals, and normalized success metrics — all over the same mesh and local `View` merge path.

---

## Leasing & editor — safe shared workspaces

Exclusive path-prefix leases let peers partition a workspace and write without stepping on each other.

| Example                               | Command                        | What it shows                                                                           |
|---------------------------------------|--------------------------------|-----------------------------------------------------------------------------------------|
| [lease_workspace](./lease_workspace/) | `make example-lease-workspace` | Path-prefix leases, advisory `ChangingError` on read, gated writes, filesystem replace. |
| [editor_mcp](./editor_mcp/)           | `make example-editor-mcp`      | An MCP SSE editor on `:3000` (set `ANIMAL_AGENT_WORKSPACE` first).                      |

---

## Alcatraz session — agents in a live Linux environment

`ai/tool/alcatraz` wraps any `io.ReadWriter` (such as `github.com/theapemachine/alcatraz/pkg/environment.Session`): environment stdout/stderr is read as agent prompt input, and assistant output is written back to stdin. It also exposes `alcatraz_read` and `alcatraz_write` MCP tools. `ai/session` binds an `ai.Agent`, a streaming provider, and the alcatraz bridge so streamed model deltas land on stdin as they arrive — and it can run an A2A task by cloning the agent, claiming a `lease_prefix` when present, and reporting completion metrics or blocker signals through swarm.

| Example                                 | Command                            | What it shows                                                                                                                             |
|-----------------------------------------|------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------|
| [alcatraz_session](./alcatraz_session/) | `make example-alcatraz-session` 🧠 | Starts a hardened alcatraz environment, attaches a live exec stream, wraps it with `ai/tool/alcatraz`, and runs one `ai/session.Cycle()`. |

> Requires **Docker** for the runnable example. Run `make test-alcatraz-session` for the nested example module's tests.

---

## Conversation — multi-agent salon

| Example                                     | Command                              | What it shows                                                                                                                     |
|---------------------------------------------|--------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------|
| [conversation_salon](./conversation_salon/) | `make example-conversation-salon` 🧠 | Sentience-panel personas with proper chat roles, a persistent moderator anchor, and distinctive-theme clustering. Ctrl+C to stop. |

---

## Coding horizon — a long-horizon AI-native coding loop

An observe → plan → mutate → prove loop that builds its own backlog (from your goal plus static repo hygiene analysis) and announces progress over the swarm.

| Example                             | Command                                                                                                             | What it shows                                                                                                                                      |
|-------------------------------------|---------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| [coding_horizon](./coding_horizon/) | `make example-coding-horizon-dry`                                                                                   | Dry run (no LLM edits, capped cycles): observe repo, build a goal + hygiene backlog, and walk recon/plan/mutate/prove cycles with swarm announces. |
| [coding_horizon](./coding_horizon/) | `go run -ldflags=-checklinkname=0 ./examples/coding_horizon -workspace /path/to/repo -goal "your one-line goal"` 🧠 | The full loop: LLM intake, atomic replace slices, package-scoped `go test` proof, and a final audit.                                               |

Pass your objective with `-goal "..."`. **Omit `-goal` for hygiene-only mode** — the loop still refactors and optimizes based on static analysis (oversized files, missing tests, mutex/channel signals).

---

## Memory — recall that doesn't pollute context

Each mode demonstrates a different layer of the [memory system](../README.md#memory).

| Example             | Command                            | What it shows                                                                                                                                   |
|---------------------|------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------|
| [memory](./memory/) | `make example-memory-dmt`          | DMT cognitive memory: durable records enter the radix forest, train episodic/REM sensory paths, and recall returns documents plus relations.    |
| [memory](./memory/) | `make example-memory-dmt-friction` | A swarm hygiene backlog: friction/quality observations are remembered, related, and recalled for agents hunting useful work.                    |
| [memory](./memory/) | `make example-memory-dmt-analog`   | Structural analog recall: a new-but-similar lease conflict recalls the closest prior task route via DMT shared-prefix search.                   |
| [memory](./memory/) | `make example-memory-projection`   | Projection-aware memory: a local projector enriches records with embeddings before DMT storage.                                                 |
| [memory](./memory/) | `make example-memory-manifold`     | Optional nomagique manifold projection: latent embeddings, energy, and surprise produced before consolidation. *(Requires darwin + cgo/Metal.)* |

---

## Shared helpers

[examples/support](./support/) resolves `cmd/cfg/config.yml`, builds a minimal qpool, and exposes the default swarm/lease options used across every example. Read it once and the rest of the examples become much easier to follow.
