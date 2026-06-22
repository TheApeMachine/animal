package ai

import (
	"strings"

	"github.com/theapemachine/datura/dmt"
	"github.com/theapemachine/datura/types"
)

const (
	dmtMemoryBeamWidth = 8
	dmtMemoryBeamHops  = 3
)

type dmtMemorySearch struct {
	store     *DMTMemoryStore
	query     types.Query
	limit     int
	tree      *dmt.Tree
	sequences [][]byte
}

func newDMTMemorySearch(
	store *DMTMemoryStore,
	query types.Query,
) *dmtMemorySearch {
	limit := query.Limit
	if limit <= 0 {
		limit = 8
	}

	return &dmtMemorySearch{
		store:     store,
		query:     query,
		limit:     limit,
		sequences: memorySuffixSequences(query.Text),
	}
}

func (search *dmtMemorySearch) validate() error {
	if err := ensureCognitiveText(search.query.Text); err != nil {
		return err
	}

	tree, err := search.store.tree()
	if err != nil {
		return err
	}

	search.tree = tree

	return nil
}

func (search *dmtMemorySearch) run() (types.Memory, error) {
	out := types.NewMemory()
	seenDocuments := make(map[string]struct{}, search.limit)
	seenRelationships := make(map[string]struct{}, search.limit)

	for _, sequence := range search.candidates() {
		search.collectDocuments(&out, seenDocuments, sequence)
		search.collectRelationships(&out, seenRelationships, sequence)

		if len(out.Documents) >= search.limit {
			break
		}
	}

	return out, nil
}

func (search *dmtMemorySearch) candidates() [][]byte {
	candidates := make([][]byte, 0, len(search.sequences)+dmtMemoryBeamWidth)
	seen := make(map[string]struct{}, len(search.sequences)+dmtMemoryBeamWidth)
	scratch := &dmt.BeamSearchScratch{
		CurrentBeams: make([]dmt.BeamPath, 0, dmtMemoryBeamWidth),
		NextBeams:    make([]dmt.BeamPath, 0, dmtMemoryBeamWidth),
		LookupBuffer: make([]dmt.LookaheadPrediction, 0, dmtMemoryBeamWidth),
	}

	for _, sequence := range search.sequences {
		search.store.forest.EvaluateCuriosityAndTriggerSync(
			[]byte(dmtMemorySensoryPrefix + string(sequence)),
		)

		candidates = appendUniqueSequence(candidates, seen, sequence)

		for _, path := range search.tree.ExecuteBeamSearch(
			sequence,
			dmtMemoryBeamWidth,
			dmtMemoryBeamHops,
			scratch,
		) {
			candidates = appendUniqueSequence(candidates, seen, path.Sequence)
		}
	}

	return candidates
}

func (search *dmtMemorySearch) collectDocuments(
	out *types.Memory,
	seen map[string]struct{},
	sequence []byte,
) {
	search.walkDocuments(out, seen, []byte(sequenceIndexPrefix(sequence)))

	if len(out.Documents) > 0 {
		return
	}

	analog, found := search.tree.FindStructuralAnalog([]byte(sequenceIndexPrefix(sequence)))
	if !found {
		return
	}

	search.walkDocuments(out, seen, analog.ClosestKey)
}

func (search *dmtMemorySearch) walkDocuments(
	out *types.Memory,
	seen map[string]struct{},
	prefix []byte,
) {
	search.tree.WalkPrefix(prefix, func(key []byte, value []byte) bool {
		documentKey := documentKeyFromSequenceIndex(key)
		if documentKey == "" {
			return true
		}

		if _, ok := seen[documentKey]; ok {
			return true
		}

		payload, found := search.store.forest.Get([]byte(documentKey))
		if !found {
			return true
		}

		record, err := decodeDMTDocument(payload)
		if err != nil || search.store.deleted(record.ID) || !search.matchesScope(record.Scope) {
			return true
		}

		seen[documentKey] = struct{}{}
		out.AddDocument(record.document())

		return len(out.Documents) < search.limit
	})
}

func (search *dmtMemorySearch) collectRelationships(
	out *types.Memory,
	seen map[string]struct{},
	sequence []byte,
) {
	search.tree.WalkPrefix([]byte(relationshipIndexPrefix(sequence)), func(key []byte, value []byte) bool {
		relationshipKey := relationshipKeyFromIndex(key)
		if relationshipKey == "" {
			return true
		}

		if _, ok := seen[relationshipKey]; ok {
			return true
		}

		payload, found := search.store.forest.Get([]byte(relationshipKey))
		if !found {
			return true
		}

		record, err := decodeDMTRelationship(payload)
		if err != nil || !search.matchesScope(record.Scope) {
			return true
		}

		seen[relationshipKey] = struct{}{}
		out.AddRelationship(record.relationship())

		return true
	})
}

func (search *dmtMemorySearch) matchesScope(scope string) bool {
	queryScope := strings.TrimSpace(search.query.Metadata.Source)

	return queryScope == "" || scope == queryScope
}

func appendUniqueSequence(
	sequences [][]byte,
	seen map[string]struct{},
	sequence []byte,
) [][]byte {
	key := string(sequence)

	if _, ok := seen[key]; ok {
		return sequences
	}

	seen[key] = struct{}{}

	return append(sequences, append([]byte(nil), sequence...))
}
