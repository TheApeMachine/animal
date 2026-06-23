package swarm

import (
	"time"

	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/errnie"
)

func (participant *Participant) reportContention(prefix string, err error) error {
	holderID, holderPrefix := participant.contentionHolder(prefix, err)
	contention, err := NewContentionAt(
		participant.actorID,
		participant.actorName,
		participant.role,
		prefix,
		err.Error(),
		time.Now(),
	)

	if err != nil {
		return err
	}

	contention.HolderID = holderID
	contention.HolderPrefix = holderPrefix

	if err := contention.Validate(); err != nil {
		return err
	}

	if err := participant.view.MergeContention(contention); err != nil {
		return err
	}

	return participant.mesh.PublishValue(
		participant.actorID,
		MessageTypeContention,
		contention,
	)
}

func (participant *Participant) contentionHolder(
	prefix string,
	err error,
) (string, string) {
	conflict, ok := lease.AsConflict(err)

	if ok {
		return conflict.ActorID, conflict.LeaseKey
	}

	holderID, ok := participant.view.ClaimHolder(prefix)

	if ok {
		return holderID, prefix
	}

	return "", ""
}

func (participant *Participant) contentionError(prefix string, err error) error {
	return errnie.Combine(err, participant.reportContention(prefix, err))
}
