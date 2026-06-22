# animal

**A Go framework for orchestrating AI agents that collaborate as peers — not as a manager-and-worker hierarchy.**

`animal` gives you the pieces to build LLM-driven agents that stream from OpenAI-compatible providers, coordinate over a gossip mesh, claim exclusive slices of a shared workspace, remember what matters across runs, and operate live inside a hardened Linux environment — all in idiomatic Go.

```go
agent, _ := ai.NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/vertical-a/"})
agent.Cycle()                                  // merge swarm gossip into context
llm.Stream(agent.System, &agent.Context, provider.NewParams())
```

---

## Why animal?

Most agent frameworks bolt a planner on top of a worker and call it a day. `animal` takes a different stance: agents are **autonomous peers** that share state, negotiate work through leases, and broadcast progress over a mesh. No single agent is in charge. This makes it a natural fit for swarms, long-horizon coding loops, and multi-agent conversations where coordination should emerge rather than be dictated.

The design favors:

- **Streaming-first** model calls through any OpenAI-compatible endpoint (LM Studio, vLLM, OpenAI, etc.).
- **Coordination without hierarchy** — gossip, leases, and A2A task broadcasts over a [`qpool`](https://github.com/theapemachine/qpool) mesh.
- **Memory that stays out of the way** — durable recall happens in a temporary context, so the agent's working context never gets polluted by retrieved text.
- **Durable, auditable artifacts** — training traces and storage all share one prefix-oriented contract across DMT, blob, and S3 backends.

---

## Quick start

### Prerequisites

- **Go 1.26+** (the [`qpool`](https://github.com/theapemachine/qpool) mesh uses `go:linkname` for zero-overhead goroutine parking; Go 1.26 requires the `-checklinkname=0` linker flag — every Makefile target sets it for you).
- An **OpenAI-compatible endpoint** for any example that calls a model. A local server such as [LM Studio](https://lmstudio.ai) or [vLLM](https://github.com/vllm-project/vllm) works well. Configure it in `cmd/cfg/config.yml` (see [Configuration](#configuration)).

### Run your first example

```sh
# Pure coordination, no model required — two agents gossip over the mesh.
make example-swarm-roadmap-gossip

# Agent.Cycle() merges gossip, then streams an LLM reply (needs an endpoint).
make example-swarm-agent-cycle
```

Run the test suite:

```sh
make test     # go test -ldflags='-checklinkname=0' -race ./...
```

### Minimal end-to-end

A complete agent: construct it, let it absorb swarm traffic with `Cycle()`, then stream a reply. (Condensed from [`examples/swarm_agent_cycle`](./examples/swarm_agent_cycle/).)

```go
ctx := context.Background()
pool := support.NewQPool(ctx)

// One shared mesh + lease coordinator for all participants.
registry, _ := swarm.NewRegistry(ctx, pool,
    support.DefaultSwarmOptions("quickstart"),
    support.DefaultLeaseOptions(),
)

// An agent is a persona (role + name), a mesh participant, and a set of
// exclusive workspace prefixes it is allowed to write to.
agent, _ := ai.NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/vertical-a/"})

// Pull any pending gossip / cooperation traffic into the agent's context.
agent.Cycle()

// Add a turn and stream a model response.
agent.Context.Messages = append(agent.Context.Messages, provider.Message{
    Role:    "user",
    Content: "Acknowledge the roadmap and state your next build step.",
})

endpoint, apiKey, model := support.OpenAIConfig()
llm, _ := provider.NewOpenAI(ctx, pool, endpoint, apiKey, model)
llm.Stream(agent.System, &agent.Context, provider.NewParams())
```

> Streamed deltas are published on a `qpool` broadcast group — subscribe to it to render tokens as they arrive. See the full example for the consumer loop.

---

## Core concepts

| Package                   | What it gives you                                                                                                                                                                                          |
|---------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **`ai`**                  | The `Agent` type: model context, tools, memory, optional swarm participation, clone-based sub-task delegation, hot context swapping, and JSONL training capture.                                           |
| **`ai/provider`**         | OpenAI-compatible streaming (Responses API) with structured outputs and explicit controls: temperature, top-p, max output tokens, parallel tool calls, reasoning effort.                                   |
| **`ai/session`**          | Binds an agent, a streaming provider, an alcatraz bridge, and an optional swarm participant into one interactive cycle or A2A task run.                                                                    |
| **`ai/tool/alcatraz`**    | Bridges a live Linux environment stream to an agent — stdout/stderr becomes prompt input, assistant output is written to stdin — and exposes `alcatraz_read` / `alcatraz_write` MCP tools.                 |
| **`swarm`**               | The coordination fabric: a qpool mesh, lease-aware participants, gossip views, A2A task broadcasts, lifecycle status events, blocker/friction/quality/opportunity signals, and normalized success metrics. |
| **`storage`**             | Artifact persistence behind one prefix contract. `DMTStore` (local radix-trie + optional WAL), `BlobStore` (Go Cloud buckets), and `NewS3Store` (S3 / S3-compatible).                                      |
| **`a2a`**                 | Protocol-compatible data models: task, message, part, artifact, streaming event, agent card.                                                                                                               |
| **`lease` / `ownership`** | Exclusive prefix coordination for shared workspaces — the mechanism that lets peers partition work safely.                                                                                                 |

### The agent lifecycle

An `ai.Agent` is constructed from a **persona** (a `role` + `name` resolved against the config's persona templates), an optional **swarm registry**, and the **lease prefixes** it may write to:

```go
func NewAgent(ctx context.Context, pool *qpool.Q[any], role, name string,
    registry *swarm.Registry, claimPrefixes []string) (*Agent, error)
```

Key methods:

| Method                       | Purpose                                                                                        |
|------------------------------|------------------------------------------------------------------------------------------------|
| `Cycle()`                    | Poll incoming gossip / cooperation traffic and merge swarm artifacts into the agent's context. |
| `Clone(ctx, subTask)`        | Fork the agent with one new user message — the basis for sub-task delegation.                  |
| `CloneWithTask(ctx, task)`   | Fork from an A2A task instruction.                                                             |
| `SwapContext(agentCtx)`      | Hot-swap the message history without rebuilding the agent.                                     |
| `UseTrainingStore(ctx, cfg)` | Open artifact-backed training capture (DMT / blob / S3) and attach it.                         |
| `Participant()`              | Return the swarm coordination surface (gossip view, announces, signals).                       |

---

## Memory

Agent memory is a datura-backed runtime surface designed to **enrich without polluting**. Around each generation the session runs two strict, structured passes:

1. **Recall** (before) — searches memory and injects a compact `MemoryPacket` into a *temporary* model context.
2. **Consolidation** (after) — extracts durable records and optional graph relationships from the response back into memory.

The agent's primary context stays task-focused — retrieved memory text never leaks into it.

| Constructor                                   | Use it for                                                                                                                                                                                             |
|-----------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `NewLocalMemory(ctx)`                         | Local runs and tests. Records live in a DMT radix forest; text commits through episodic buffers; REM consolidation trains sensory paths; recall uses sensory beam search plus structural analogs.      |
| `NewDaturaMemoryWithGraph(ctx, store, graph)` | Composing non-DMT backends explicitly. Accepts separate document/vector and graph stores. *(Relationships without a graph store fail validation.)*                                                     |
| `NewProjectedMemory(ctx, memory, projector)`  | Wrapping any backend with a projection layer.                                                                                                                                                          |
| `NewManifoldProjector(ctx, cfg)`              | Projecting memory text into latent embeddings via nomagique's Metal-backed resonance manifold, exposing energy/surprise signals before consolidation reaches storage. *(Requires darwin + cgo/Metal.)* |

See [`examples/memory`](./examples/memory/) for runnable demonstrations of each mode.

---

## Storage & training capture

Artifact-backed training capture can be opened through `storage.Config`, `ai.NewTrainingStoreFromConfig`, or `agent.UseTrainingStore`. Three drivers are supported — `dmt`, `blob`, and `s3` — and all preserve the prefix contract used by training exports:

```
training/<goal>/<actor>/<timestamp>/<uuid>.jsonl
```

This means a trace captured locally during development and one streamed to S3 in production are addressable the same way.

---

## Configuration

Configuration is loaded with viper, with `cmd/cfg/config.yml` as the embedded default. Key fields:

```yaml
ai:
  model:    "openai/gpt-oss-20b"
  endpoint: "http://localhost:1234/v1"   # any OpenAI-compatible server
  apiKey:   ${OPENAI_API_KEY}            # expanded from the environment

  lease:
    idle_ttl_seconds: 900

  swarm:
    enabled: true
    mesh_id: swarm
    gossip_ttl_seconds: 30

  prompt:
    template:                            # system / observation / recall / consolidation prompts
      system: |
        You are {{ agent.name }}, a {{ agent.role }} for {{ project.name }}.

  personas:                              # role → characteristics, responsibilities, guidelines
    developer: { ... }
  workflows:                             # multi-stage persona orchestration
    roadmap_delivery: { ... }
```

Relevant environment variables:

| Variable                 | Purpose                                                                                        |
|--------------------------|------------------------------------------------------------------------------------------------|
| `OPENAI_API_KEY`         | Expanded into `ai.apiKey`.                                                                     |
| `ANIMAL_AGENT_WORKSPACE` | Agent workspace root (falls back to `ALCATRAZ_AGENT_WORKSPACE`, then sensible local defaults). |

---

## Examples

A full catalog of runnable programs — swarm coordination, leasing, the alcatraz Linux bridge, multi-agent conversation, the long-horizon coding loop, and every memory mode — lives in **[examples/README.md](./examples/README.md)**. Start there for hands-on onboarding.

---

## Roadmap

A living view of where `animal` is and where it could go. `[x]` ships today, `[~]` is partially built (scaffolding exists, wiring doesn't), and `[ ]` is a direction we're considering — feedback and PRs welcome. These are ideas for a first pass, not commitments.

### Foundations — agents & providers

- [x] `ai.Agent` with persona, context, tools, memory, and swarm participation
- [x] Clone-based sub-task delegation and hot context swapping
- [x] OpenAI-compatible streaming with structured outputs and explicit model controls
- [ ] First-class adapters for non-OpenAI providers (Anthropic, local llama.cpp, Ollama) behind the streaming interface
- [ ] Token/cost accounting surfaced per agent and per swarm run
- [ ] Retry, backoff, and graceful degradation when an endpoint stalls mid-stream
- [~] Hot-pluggable prompt templates — templates are centralized in config and `viper.WatchConfig()` is wired; needs an `OnConfigChange` handler that re-renders live agent system/observation/recall prompts without a restart

### Coordination — swarm, leases, A2A

- [x] qpool gossip mesh with lease-aware participants and conflicting-claim rejection
- [x] A2A task broadcasts, lifecycle events, and friction/quality/opportunity signals
- [x] Normalized success metrics over the mesh
- [~] Workflow orchestrator — the YAML schema (`workflows`, steps, slots, stop conditions), config accessors, and an `ai/workflow.go` stub exist; the execution engine that runs stages, spawns agents per slot, gates on file leases, and honors `stop` conditions is not yet wired (only the bespoke `coding_horizon` orchestrator runs today)
- [ ] **Heartbeat-renewed leases** — holders re-assert a prefix periodically; on missed heartbeats the lease auto-releases *with a gossip event* so a peer can pick the lane up. Turns an agent crash from "lane frozen for the idle TTL" into a non-event, and gives long tasks a clean handoff across restarts.
- [ ] **Idempotent task claims** (claim-then-confirm) — peers racing for the same A2A task before gossip converges optimistically claim, then a brief confirmation window lets the loser back off; work starts only after confirmation. Same instinct as the existing lease-conflict rejection, applied to tasks.
- [ ] **Backoff-on-contention** — a rejected claimant waits with jittered backoff instead of immediately retrying, preventing claim/release thrash on hot prefixes.
- [ ] Mesh transport beyond in-process qpool (NATS / gRPC) for cross-host swarms
- [ ] A web/TUI dashboard for live gossip, claims, and task status
- [ ] Deadlock / starvation detection when peers contend for overlapping prefixes

### Reliability & resilience

Treat the model's output as a *proposal*, and tools/peers as the *verification layer* — the philosophy already visible in editor `replace`-on-ambiguity and the coding-horizon `go test` proof step, made systematic.

- [ ] **Stream-level resilience** — detect a stalled SSE stream (no token within N seconds), cancel cleanly, and retry on the same context with a defined disposition for partial output (discard vs. keep-as-prefix) so retries never duplicate half a response.
- [ ] **Poison-task quarantine** — track per-task attempt counts in the mesh `View`; after K failures, quarantine the task and emit a blocker signal instead of letting it cycle the swarm forever.
- [ ] **Progress monitor / doom-loop breaker** — detect no-net-change across N cycles (e.g. edit → test-fail → revert → repeat) and force escalation (blocker + ask-for-help) rather than looping.
- [ ] **Verify-against-ground-truth** — a task is marked complete only when the agent's *claims* are checked against actual tool return values, not the model's narration of what it did.
- [ ] **Structured "can't proceed" primitive** — a first-class blocker output with a typed reason (missing tool, missing lease, ambiguous goal) so being stuck is an expected, observable result rather than silent hallucinated progress.
- [ ] **Context compaction** — periodically summarize-and-`SwapContext` so the working context stays task-focused over long horizons without drift or pollution.
- [ ] Bounded clone depth, tool-call loops, and gossip buffers with observable drop metrics (runaway-resource backstops).

### Agent intelligence & autonomy

Advanced, differentiating capabilities — most build directly on surfaces that already exist.

- [ ] **Peer-review channel** — a second agent scores a peer's output against the original goal *before* it's marked done, using the existing friction/quality/opportunity signals. Cheap adversarial check, no hierarchy required.
- [ ] **Shadow / counterfactual agent** — a cheap second model on the same context whose only job is to disagree: flag risky edits, question the plan, surface the failure mode the primary is about to hit. A built-in adversary as just another participant.
- [ ] **Emergent role negotiation** — agents negotiate who takes what based on advertised capabilities and current load over the gossip mesh (task-auction style), instead of personas being statically assigned from config. The opportunity/friction signals are the raw material.
- [ ] **Memory-driven analogical planning** — before starting, an agent recalls "the last time the swarm faced something structurally like this, here's what worked/failed," promoting the existing DMT structural-analog recall from a memory feature to a planning feature.
- [ ] **Surprise as a runtime control signal** — wire the nomagique manifold's energy/surprise output into agent control flow (e.g. escalate reasoning effort, slow down, or request review when surprise is high), not just into pre-consolidation storage.

### Memory & learning

- [x] DMT cognitive memory (radix forest, episodic/REM, sensory beam search, structural analogs)
- [x] Projection layer and optional nomagique Metal-backed resonance manifold
- [x] Recall/consolidation passes that keep retrieved text out of the working context
- [ ] Shared swarm memory — peers consolidate into and recall from a common store, so a lesson learned by one (a friction point, a fix that worked) is recallable by all and the swarm gets *collectively* smarter over a run (pairs with analogical planning under Agent intelligence)
- [ ] Forgetting policies (TTL, relevance decay, capped footprint)
- [ ] Memory introspection tooling (inspect/visualize what an agent recalled and why)
- [ ] Benchmarks comparing recall quality across DMT / projected / manifold backends

### Tools & environments

- [x] Three real MCP servers on `modelcontextprotocol/go-sdk`: alcatraz (`alcatraz_read`/`alcatraz_write`), editor (`read_file`/`search`/`replace`, lease-gated), browser (`browser_navigate`/`evaluate`/`content`/`click`/`wait`)
- [x] In-memory MCP client integration (`ai/mcpclient`) and per-agent tool attachment via the runner
- [ ] A shared tool registry / discovery layer — today each tool is a standalone package wired ad-hoc per call site; no central inventory or factory
- [ ] A documented tool-authoring guide so third parties can add MCP tools to the library
- [ ] Sandboxed tool execution with per-tool permission scoping
- [ ] Richer browser tool (form fill, screenshots, structured extraction)

### Training & artifacts

- [x] JSONL training capture across `dmt` / `blob` / `s3` behind one prefix contract
- [ ] An export → fine-tune → reload loop demonstrated end-to-end as an example
- [ ] Trace replay: re-run a captured session deterministically for debugging
- [ ] Quality filtering / labeling of captured traces before export

### Developer experience

- [x] Runnable examples for every major surface, with offline vs. model-required marked
- [ ] A `cmd` CLI to scaffold, run, and inspect agents/swarms without writing Go
- [ ] CI matrix and published coverage
- [ ] Godoc-grade package docs and an architecture deep-dive document
- [ ] A 5-minute "build your own agent" tutorial

---

## Verification

```sh
make test
```

runs the full suite with the race detector and the required linkname flag:

```sh
go test -ldflags='-checklinkname=0' -race ./...
```
