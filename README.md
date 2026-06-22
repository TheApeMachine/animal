# animal

`animal` is a generic Go package for AI agent orchestration. It combines provider streaming, MCP-compatible tools, qpool-backed broadcasts, lease-aware collaboration, and swarm state sharing without imposing a manager/reviewer hierarchy.

## Core Surfaces

- `ai.Agent` owns model context, tools, memory, optional swarm participation, clone-based sub-task delegation, hot context swapping, and optional JSONL training capture through file or artifact storage.
- `ai/provider` wraps OpenAI-compatible streaming Responses API calls with structured outputs and explicit model controls such as temperature, top-p, max output tokens, parallel tool calls, and reasoning effort.
- `ai/session` binds an agent, streaming provider, alcatraz bridge, and optional swarm participant into one interactive cycle or A2A task run.
- `ai/tool/alcatraz` bridges an interactive Linux environment stream, such as `alcatraz/pkg/environment.Session`, to an agent: stdout/stderr becomes prompt input, and assistant output is written to stdin. The bridge also exposes MCP tools for read/write access.
- `swarm` provides a qpool mesh, lease-aware participants, gossip views, A2A task broadcasts, task lifecycle status events, blocker/friction/quality/opportunity signals, and normalized success metrics.
- `storage` persists datura artifacts behind a prefix-oriented contract. `DMTStore` wraps `datura/dmt` for local radix-trie lookup, prefix listing, and optional WAL persistence; `BlobStore` wraps Go Cloud buckets, with `NewS3Store` using datura's S3 client for S3 and S3-compatible object storage.
- `a2a` defines protocol-compatible task, message, part, artifact, streaming event, and agent-card data models.
- `lease` and `ownership` provide exclusive prefix coordination for shared workspaces.

## Memory

Agent memory is a datura-backed runtime surface. Before a generation, the session runs a strict structured recall pass, searches memory, and injects the compact `MemoryPacket` into a temporary model context. After the assistant response, a strict consolidation pass extracts durable records and optional graph relationships back into memory. The agent's primary context remains task-focused and is not polluted by retrieved memory text.

`ai.NewLocalMemory` uses DMT cognitive memory for local runs and tests: records are stored in the radix forest, text is committed through episodic buffers, REM consolidation trains sensory paths, and recall uses sensory beam search plus structural analogs. `ai.NewDaturaMemoryWithGraph` still accepts separate document/vector and graph stores so non-DMT backends can be composed explicitly; relationships without a graph store fail validation.

`ai.NewProjectedMemory` can wrap any memory backend with a projection layer. `ai.NewManifoldProjector` uses nomagique's Metal-backed resonance manifold to project memory text into latent embeddings and expose energy/surprise signals before consolidation reaches storage.

## Storage

Artifact-backed training capture can be opened through `storage.Config`, `ai.NewTrainingStoreFromConfig`, or `agent.UseTrainingStore`.
Supported drivers are `dmt`, `blob`, and `s3`; all preserve the prefix contract used by training exports such as `training/<goal>/<actor>/<timestamp>/<uuid>.jsonl`.

## Verification

Go 1.26+ needs the qpool linkname flag. Use the Makefile:

```sh
make test
```

The target runs:

```sh
go test -ldflags='-checklinkname=0' -race ./...
```
