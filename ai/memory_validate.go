package ai

import (
	"strings"

	"github.com/theapemachine/errnie"
)

/*
Validate checks the recall plan.
*/
func (plan MemoryRecallPlan) Validate() error {
	for _, query := range plan.Queries {
		if err := query.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Validate checks one memory query.
*/
func (query MemoryQuery) Validate() error {
	if strings.TrimSpace(query.ID) == "" &&
		strings.TrimSpace(query.Text) == "" &&
		len(query.Embedding) == 0 {
		return errnie.Err(errnie.Validation, "memory query text or ID is required", nil)
	}

	if query.Limit <= 0 {
		return errnie.Err(errnie.Validation, "memory query limit is required", nil)
	}

	return nil
}

/*
Validate checks consolidation output.
*/
func (consolidation MemoryConsolidation) Validate() error {
	for _, record := range consolidation.Records {
		if err := record.Validate(); err != nil {
			return err
		}
	}

	for _, relationship := range consolidation.Relationships {
		if err := relationship.Validate(); err != nil {
			return err
		}
	}

	return nil
}

/*
Validate checks one memory record.
*/
func (record MemoryRecord) Validate() error {
	if strings.TrimSpace(record.ID) == "" {
		return errnie.Err(errnie.Validation, "memory record ID is required", nil)
	}

	if strings.TrimSpace(record.Scope) == "" {
		return errnie.Err(errnie.Validation, "memory record scope is required", nil)
	}

	if strings.TrimSpace(record.Text) == "" {
		return errnie.Err(errnie.Validation, "memory record text is required", nil)
	}

	if record.Importance < 0 || record.Importance > 1 {
		return errnie.Err(errnie.Validation, "memory record importance must be between 0 and 1", nil)
	}

	return nil
}

/*
Validate checks one memory relationship.
*/
func (relationship MemoryRelationship) Validate() error {
	if strings.TrimSpace(relationship.ID) == "" {
		return errnie.Err(errnie.Validation, "memory relationship ID is required", nil)
	}

	if strings.TrimSpace(relationship.Scope) == "" {
		return errnie.Err(errnie.Validation, "memory relationship scope is required", nil)
	}

	if strings.TrimSpace(relationship.FromID) == "" {
		return errnie.Err(errnie.Validation, "memory relationship source ID is required", nil)
	}

	if strings.TrimSpace(relationship.ToID) == "" {
		return errnie.Err(errnie.Validation, "memory relationship target ID is required", nil)
	}

	if strings.TrimSpace(relationship.Relationship) == "" {
		return errnie.Err(errnie.Validation, "memory relationship type is required", nil)
	}

	if relationship.Importance < 0 || relationship.Importance > 1 {
		return errnie.Err(errnie.Validation, "memory relationship importance must be between 0 and 1", nil)
	}

	return nil
}
