package swarm

import (
	"fmt"
	"strings"
	"time"

	"github.com/theapemachine/errnie"
)

/*
Contention records one failed attempt to acquire a lease prefix.
*/
type Contention struct {
	ActorID      string
	ActorName    string
	Role         string
	Prefix       string
	HolderID     string
	HolderPrefix string
	Error        string
	At           int64
}

/*
NewContentionAt builds a contention event with explicit time.
*/
func NewContentionAt(
	actorID string,
	actorName string,
	role string,
	prefix string,
	errText string,
	at time.Time,
) (Contention, error) {
	contention := Contention{
		ActorID:   actorID,
		ActorName: actorName,
		Role:      role,
		Prefix:    prefix,
		Error:     errText,
		At:        at.UnixNano(),
	}

	return contention, contention.Validate()
}

/*
Validate checks required contention event fields.
*/
func (contention Contention) Validate() error {
	if strings.TrimSpace(contention.ActorID) == "" {
		return errnie.Err(errnie.Validation, "swarm contention actor id is required", nil)
	}

	if strings.TrimSpace(contention.Prefix) == "" {
		return errnie.Err(errnie.Validation, "swarm contention prefix is required", nil)
	}

	if contention.At <= 0 {
		return errnie.Err(errnie.Validation, "swarm contention timestamp is required", nil)
	}

	if strings.TrimSpace(contention.Error) == "" {
		return errnie.Err(errnie.Validation, "swarm contention error is required", nil)
	}

	if contention.HolderID == contention.ActorID {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm contention holder matches actor %q", contention.ActorID),
			nil,
		)
	}

	return nil
}
