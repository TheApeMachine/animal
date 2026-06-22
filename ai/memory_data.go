package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/theapemachine/datura/types"
)

func (packet *MemoryPacket) merge(result types.Memory, scope string) {
	for _, document := range result.Documents {
		documentScope := document.Metadata.Source

		if strings.TrimSpace(scope) != "" && documentScope != scope {
			continue
		}

		memoryDocument := MemoryDocument{
			ID:        document.ID,
			Scope:     documentScope,
			Text:      document.Text,
			Embedding: append([]float32(nil), document.Embedding...),
		}

		if packet.hasDocument(memoryDocument) {
			continue
		}

		packet.Documents = append(packet.Documents, memoryDocument)
	}

	for _, relationship := range result.Relationships {
		relationshipScope := relationship.Metadata.Source

		if strings.TrimSpace(scope) != "" && relationshipScope != scope {
			continue
		}

		memoryRelationship := MemoryRelationship{
			ID:           relationship.Metadata.ID,
			Scope:        relationshipScope,
			FromID:       relationship.ID,
			ToID:         relationship.ToID,
			Relationship: relationship.Relationship,
		}

		if packet.hasRelationship(memoryRelationship) {
			continue
		}

		packet.Relationships = append(packet.Relationships, memoryRelationship)
	}
}

func (packet MemoryPacket) hasDocument(document MemoryDocument) bool {
	for _, existing := range packet.Documents {
		if existing.Scope == document.Scope && existing.ID == document.ID {
			return true
		}
	}

	return false
}

func (packet MemoryPacket) hasRelationship(relationship MemoryRelationship) bool {
	for _, existing := range packet.Relationships {
		if existing.Scope == relationship.Scope && existing.ID == relationship.ID {
			return true
		}
	}

	return false
}

func (query MemoryQuery) datura() types.Query {
	return types.Query{
		ID:        strings.TrimSpace(query.ID),
		Text:      strings.TrimSpace(query.Text),
		Embedding: append([]float32(nil), query.Embedding...),
		Metadata: types.Metadata{
			Source: strings.TrimSpace(query.Scope),
		},
		Limit:        query.Limit,
		VectorWeight: query.VectorWeight,
		TextWeight:   query.TextWeight,
	}
}

func (record MemoryRecord) mutation() types.Mutation {
	return types.Mutation{
		ID:        strings.TrimSpace(record.ID),
		Text:      strings.TrimSpace(record.Text),
		Embedding: append([]float32(nil), record.Embedding...),
		Metadata: types.Metadata{
			ID:        strings.TrimSpace(record.ID),
			Source:    strings.TrimSpace(record.Scope),
			Timestamp: time.Now().UTC(),
		},
	}
}

func (relationship MemoryRelationship) mutation() types.Mutation {
	return types.Mutation{
		ID:           strings.TrimSpace(relationship.FromID),
		Relationship: strings.TrimSpace(relationship.Relationship),
		RelatedID:    strings.TrimSpace(relationship.ToID),
		Metadata: types.Metadata{
			ID:        strings.TrimSpace(relationship.ID),
			Source:    strings.TrimSpace(relationship.Scope),
			Timestamp: time.Now().UTC(),
		},
	}
}

/*
Format renders memory for injection into a model generation.
*/
func (packet MemoryPacket) Format() string {
	var builder strings.Builder

	if len(packet.Documents) == 0 && len(packet.Relationships) == 0 {
		return ""
	}

	builder.WriteString("Relevant memory:\n")

	for _, document := range packet.Documents {
		builder.WriteString(fmt.Sprintf(
			"- [%s/%s] %s\n",
			document.Scope,
			document.ID,
			document.Text,
		))
	}

	for _, relationship := range packet.Relationships {
		builder.WriteString(fmt.Sprintf(
			"- [%s/%s] %s -%s-> %s\n",
			relationship.Scope,
			relationship.ID,
			relationship.FromID,
			relationship.Relationship,
			relationship.ToID,
		))
	}

	return strings.TrimSpace(builder.String())
}
