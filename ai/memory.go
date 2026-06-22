package ai

import (
	"context"
	"strings"

	"github.com/theapemachine/datura/types"
	"github.com/theapemachine/errnie"
)

/*
Memory is a runtime recall and consolidation surface attached to an agent.
*/
type Memory interface {
	Recall(ctx context.Context, plan MemoryRecallPlan) (MemoryPacket, error)
	Remember(ctx context.Context, consolidation MemoryConsolidation) error
	Forget(ctx context.Context, ids []string) error
	Close() error
}

/*
MemoryQuery is one runtime search request produced by the recall pass.
*/
type MemoryQuery struct {
	ID           string    `json:"id"`
	Scope        string    `json:"scope"`
	Text         string    `json:"text"`
	Embedding    []float32 `json:"embedding,omitempty"`
	Limit        int       `json:"limit"`
	VectorWeight float64   `json:"vector_weight"`
	TextWeight   float64   `json:"text_weight"`
}

/*
MemoryRecallPlan is the structured output of the memory recall pass.
*/
type MemoryRecallPlan struct {
	Queries []MemoryQuery `json:"queries"`
}

/*
MemoryDocument is one document recalled from memory.
*/
type MemoryDocument struct {
	ID        string    `json:"id"`
	Scope     string    `json:"scope"`
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding,omitempty"`
}

/*
MemoryRelationship is one graph edge remembered by a graph-capable backend.
*/
type MemoryRelationship struct {
	ID           string  `json:"id"`
	Scope        string  `json:"scope"`
	FromID       string  `json:"from_id"`
	ToID         string  `json:"to_id"`
	Relationship string  `json:"relationship"`
	Importance   float64 `json:"importance"`
}

/*
MemoryPacket is the compact recalled memory injected into a generation.
*/
type MemoryPacket struct {
	Documents     []MemoryDocument     `json:"documents"`
	Relationships []MemoryRelationship `json:"relationships"`
}

/*
MemoryRecord is one document requested by the consolidation pass.
*/
type MemoryRecord struct {
	ID         string    `json:"id"`
	Scope      string    `json:"scope"`
	Text       string    `json:"text"`
	Embedding  []float32 `json:"embedding,omitempty"`
	Importance float64   `json:"importance"`
}

/*
MemoryConsolidation is the structured output of the memory consolidation pass.
*/
type MemoryConsolidation struct {
	Records       []MemoryRecord       `json:"records"`
	Relationships []MemoryRelationship `json:"relationships"`
	Forget        []string             `json:"forget"`
}

/*
DaturaMemory adapts datura's unified Store interface to agent memory.
*/
type DaturaMemory struct {
	ctx    context.Context
	cancel context.CancelFunc
	err    error
	store  types.Store
	graph  types.Store
}

/*
NewDaturaMemory instantiates memory over one datura document or vector store.
*/
func NewDaturaMemory(ctx context.Context, store types.Store) (*DaturaMemory, error) {
	return NewDaturaMemoryWithGraph(ctx, store, nil)
}

/*
NewLocalMemory instantiates in-process datura memory for tests and local runs.
*/
func NewLocalMemory(ctx context.Context) (*DaturaMemory, error) {
	store, err := NewDMTMemoryStore(ctx, DMTMemoryConfig{})
	if err != nil {
		return nil, err
	}

	return NewDaturaMemoryWithGraph(ctx, store, store)
}

/*
NewDaturaMemoryWithGraph instantiates memory over document/vector and graph stores.
*/
func NewDaturaMemoryWithGraph(
	ctx context.Context,
	store types.Store,
	graph types.Store,
) (*DaturaMemory, error) {
	if ctx == nil {
		return nil, errnie.Err(errnie.Validation, "memory context is required", nil)
	}

	if store == nil {
		return nil, errnie.Err(errnie.Validation, "memory store is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	memory := &DaturaMemory{
		ctx:    ctx,
		cancel: cancel,
		store:  store,
		graph:  graph,
	}

	return memory, errnie.Require(map[string]any{
		"ctx":    memory.ctx,
		"cancel": memory.cancel,
		"store":  memory.store,
	})
}

/*
Recall searches memory using the recall plan.
*/
func (memory *DaturaMemory) Recall(
	ctx context.Context,
	plan MemoryRecallPlan,
) (MemoryPacket, error) {
	if err := plan.Validate(); err != nil {
		return MemoryPacket{}, err
	}

	packet := MemoryPacket{
		Documents:     make([]MemoryDocument, 0),
		Relationships: make([]MemoryRelationship, 0),
	}

	for _, query := range plan.Queries {
		result, err := memory.query(ctx, query)
		if err != nil {
			return MemoryPacket{}, err
		}

		packet.merge(result, query.Scope)
	}

	return packet, nil
}

/*
Remember writes consolidation output to memory.
*/
func (memory *DaturaMemory) Remember(
	ctx context.Context,
	consolidation MemoryConsolidation,
) error {
	if err := consolidation.Validate(); err != nil {
		return err
	}

	for _, record := range consolidation.Records {
		if err := memory.store.Put(ctx, record.mutation()); err != nil {
			return errnie.Err(errnie.IO, "memory record write failed", err)
		}
	}

	if len(consolidation.Relationships) > 0 && memory.graph == nil {
		return errnie.Err(errnie.Validation, "memory graph store is required", nil)
	}

	for _, relationship := range consolidation.Relationships {
		if err := memory.graph.Put(ctx, relationship.mutation()); err != nil {
			return errnie.Err(errnie.IO, "memory relationship write failed", err)
		}
	}

	return memory.Forget(ctx, consolidation.Forget)
}

/*
Forget deletes memories by ID.
*/
func (memory *DaturaMemory) Forget(ctx context.Context, ids []string) error {
	for _, id := range ids {
		id = strings.TrimSpace(id)

		if id == "" {
			return errnie.Err(errnie.Validation, "memory forget ID is required", nil)
		}

		if err := memory.store.Delete(ctx, types.Mutation{ID: id}); err != nil {
			return errnie.Err(errnie.IO, "memory forget failed", err)
		}
	}

	return nil
}

/*
Close closes store resources when the backend exposes a Close method.
*/
func (memory *DaturaMemory) Close() error {
	memory.cancel()

	if closer, ok := memory.store.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}

	if closer, ok := memory.graph.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

func (memory *DaturaMemory) query(
	ctx context.Context,
	query MemoryQuery,
) (types.Memory, error) {
	if strings.TrimSpace(query.ID) != "" {
		return memory.store.Get(ctx, query.datura())
	}

	result, err := memory.store.Search(ctx, query.datura())
	if err != nil {
		return types.Memory{}, errnie.Err(errnie.IO, "memory search failed", err)
	}

	return result, nil
}
