package swarm

import (
	"fmt"
	"time"
)

/*
Rumor is a typed gossip frame exchanged between agents on the mesh.
*/
type Rumor struct {
	Kind      Kind
	ActorID   string
	ActorName string
	Role      string
	Prefix    string
	Topic     string
	Payload   string
	State     string
	At        int64
}

/*
NewRumorAt builds a rumor stamped with now.
*/
func NewRumorAt(
	kind Kind,
	actorID, actorName, role string,
	at time.Time,
) Rumor {
	return Rumor{
		Kind:      kind,
		ActorID:   actorID,
		ActorName: actorName,
		Role:      role,
		At:        at.UnixNano(),
	}
}

/*
Validate checks required fields for the rumor kind.
*/
func (rumor Rumor) Validate() error {
	if rumor.Kind == "" {
		return fmt.Errorf("swarm: rumor kind is required")
	}

	if rumor.ActorID == "" {
		return fmt.Errorf("swarm: rumor actor ID is required")
	}

	if rumor.At <= 0 {
		return fmt.Errorf("swarm: rumor timestamp is required")
	}

	switch rumor.Kind {
	case KindAnnounce:
		if rumor.Topic == "" {
			return fmt.Errorf("swarm: announce topic is required")
		}
	case KindClaim, KindRelease:
		if rumor.Prefix == "" {
			return fmt.Errorf("swarm: claim/release prefix is required")
		}
	case KindStatus:
		if rumor.State == "" {
			return fmt.Errorf("swarm: status state is required")
		}
	default:
		return fmt.Errorf("swarm: unknown rumor kind %q", rumor.Kind)
	}

	return nil
}
