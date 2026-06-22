package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/theapemachine/datura/types"
	"github.com/theapemachine/errnie"
)

type dmtDocumentRecord struct {
	ID        string         `json:"id"`
	Scope     string         `json:"scope"`
	Text      string         `json:"text"`
	Embedding []float32      `json:"embedding,omitempty"`
	Metadata  types.Metadata `json:"metadata"`
}

func newDMTDocumentRecord(mutation types.Mutation) (dmtDocumentRecord, error) {
	id := strings.TrimSpace(mutation.ID)
	scope := strings.TrimSpace(mutation.Metadata.Source)
	text := strings.TrimSpace(mutation.Text)

	if id == "" {
		return dmtDocumentRecord{}, errnie.Err(errnie.Validation, "dmt memory document ID is required", nil)
	}

	if scope == "" {
		return dmtDocumentRecord{}, errnie.Err(errnie.Validation, "dmt memory document scope is required", nil)
	}

	if err := ensureCognitiveText(text); err != nil {
		return dmtDocumentRecord{}, err
	}

	return dmtDocumentRecord{
		ID:        id,
		Scope:     scope,
		Text:      text,
		Embedding: append([]float32(nil), mutation.Embedding...),
		Metadata:  mutation.Metadata,
	}, nil
}

func decodeDMTDocument(payload []byte) (dmtDocumentRecord, error) {
	var record dmtDocumentRecord

	if err := json.Unmarshal(payload, &record); err != nil {
		return dmtDocumentRecord{}, errnie.Err(errnie.Validation, "dmt memory document decode failed", err)
	}

	return record, nil
}

func (record dmtDocumentRecord) documentKey() string {
	return documentKey(record.Scope, record.ID)
}

func (record dmtDocumentRecord) document() types.Document {
	return types.Document{
		ID:        record.ID,
		Text:      record.Text,
		Embedding: append([]float32(nil), record.Embedding...),
		Metadata:  record.Metadata,
	}
}

type dmtRelationshipRecord struct {
	ID           string         `json:"id"`
	Scope        string         `json:"scope"`
	FromID       string         `json:"from_id"`
	ToID         string         `json:"to_id"`
	Relationship string         `json:"relationship"`
	Importance   float64        `json:"importance"`
	Metadata     types.Metadata `json:"metadata"`
}

func newDMTRelationshipRecord(mutation types.Mutation) (dmtRelationshipRecord, error) {
	record := dmtRelationshipRecord{
		ID:           strings.TrimSpace(mutation.Metadata.ID),
		Scope:        strings.TrimSpace(mutation.Metadata.Source),
		FromID:       strings.TrimSpace(mutation.ID),
		ToID:         strings.TrimSpace(mutation.RelatedID),
		Relationship: strings.TrimSpace(mutation.Relationship),
		Importance:   1,
		Metadata:     mutation.Metadata,
	}

	if record.ID == "" {
		record.ID = fmt.Sprintf("%s:%s:%s", record.FromID, record.Relationship, record.ToID)
	}

	return record, record.validate()
}

func (record dmtRelationshipRecord) validate() error {
	if record.Scope == "" {
		return errnie.Err(errnie.Validation, "dmt memory relationship scope is required", nil)
	}

	if record.FromID == "" || record.ToID == "" {
		return errnie.Err(errnie.Validation, "dmt memory relationship endpoints are required", nil)
	}

	if record.Relationship == "" {
		return errnie.Err(errnie.Validation, "dmt memory relationship type is required", nil)
	}

	return nil
}

func decodeDMTRelationship(payload []byte) (dmtRelationshipRecord, error) {
	var record dmtRelationshipRecord

	if err := json.Unmarshal(payload, &record); err != nil {
		return dmtRelationshipRecord{}, errnie.Err(errnie.Validation, "dmt memory relationship decode failed", err)
	}

	return record, nil
}

func (record dmtRelationshipRecord) relationshipKey() string {
	return relationshipKey(record.Scope, record.ID)
}

func (record dmtRelationshipRecord) relationship() types.Relationship {
	return types.Relationship{
		ID:           record.FromID,
		Relationship: record.Relationship,
		ToID:         record.ToID,
		Metadata:     record.Metadata,
	}
}

func documentKey(scope string, id string) string {
	return dmtMemoryDocumentPrefix + memoryPath(scope) + "/" + memoryPath(id)
}

func documentIndexKey(id string) string {
	return dmtMemoryDocumentIndex + memoryPath(id)
}

func sequenceIndexPrefix(sequence []byte) string {
	return dmtMemorySequenceIndex + string(sequence) + "/"
}

func sequenceIndexKey(sequence []byte, documentKey string) string {
	return sequenceIndexPrefix(sequence) + documentKey
}

func tombstoneKey(id string) string {
	return dmtMemoryTombstonePrefix + memoryPath(id)
}

func relationshipKey(scope string, id string) string {
	return dmtMemoryRelationshipPrefix + memoryPath(scope) + "/" + memoryPath(id)
}

func relationshipIndexPrefix(sequence []byte) string {
	return dmtMemoryRelationshipIndex + string(sequence) + "/"
}

func relationshipIndexKey(sequence []byte, relationshipKey string) string {
	return relationshipIndexPrefix(sequence) + relationshipKey
}

func documentKeyFromSequenceIndex(key []byte) string {
	text := string(key)
	parts := strings.SplitN(text, "/", 5)

	if len(parts) < 5 {
		return ""
	}

	return parts[4]
}

func relationshipKeyFromIndex(key []byte) string {
	text := string(key)
	parts := strings.SplitN(text, "/", 5)

	if len(parts) < 5 {
		return ""
	}

	return parts[4]
}

func memorySuffixSequences(text string) [][]byte {
	tokens := memoryTokens(text)
	sequences := make([][]byte, 0, len(tokens))

	for index := range tokens {
		sequence := []byte(strings.Join(tokens[index:], "_"))
		if len(sequence) == 0 {
			continue
		}

		sequences = append(sequences, sequence)
	}

	return sequences
}

func memorySequence(text string) []byte {
	tokens := memoryTokens(text)

	return []byte(strings.Join(tokens, "_"))
}

func memoryTokens(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}

func memoryToken(text string) string {
	tokens := memoryTokens(text)

	if len(tokens) == 0 {
		return "related"
	}

	return strings.Join(tokens, "_")
}

func memoryPath(value string) string {
	var builder strings.Builder

	for _, character := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(character) || unicode.IsNumber(character) {
			builder.WriteRune(character)

			continue
		}

		builder.WriteByte('_')
	}

	return strings.Trim(builder.String(), "_")
}

var errDMTMemoryEmptyText = errors.New("dmt memory text produced no cognitive sequence")

func ensureCognitiveText(text string) error {
	if len(memoryTokens(text)) == 0 {
		return errnie.Err(errnie.Validation, errDMTMemoryEmptyText.Error(), nil)
	}

	return nil
}
