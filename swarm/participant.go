package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Participant is one agent's gossip + lease coordination surface.
*/
type Participant struct {
	ctx           context.Context
	cancel        context.CancelFunc
	err           error
	actorID       string
	actorName     string
	role          string
	claimPrefixes []string
	view          *View
	mesh          *Mesh
	coordinator   *lease.Coordinator
	subscriber    *qpool.BroadcastConsumer
}

/*
View returns the participant's merged situational picture.
*/
func (participant *Participant) View() *View {
	return participant.view
}

/*
Incoming blocks until the next mesh rumor is available.
*/
func (participant *Participant) Incoming() (*datura.Artifact, error) {
	return participant.subscriber.Wait(participant.ctx)
}

/*
Poll returns the next pending mesh rumor without blocking.
*/
func (participant *Participant) Poll() *datura.Artifact {
	if participant.subscriber == nil {
		return nil
	}

	return participant.subscriber.Poll()
}

/*
Receive merges an inbound rumor into the local view.
*/
func (participant *Participant) Receive(rumor Rumor) error {
	participant.view.PurgeExpired(time.Now())

	return participant.view.Merge(rumor)
}

/*
ReceiveArtifact merges any typed mesh artifact into the local view.
*/
func (participant *Participant) ReceiveArtifact(artifact *datura.Artifact) error {
	if artifact == nil {
		return errnie.Err(errnie.Validation, "swarm artifact is required", nil)
	}

	participant.view.PurgeExpired(time.Now())

	switch qpool.BusMessageType(artifact) {
	case MessageTypeRumor:
		return participant.view.Merge(datura.As[Rumor](artifact))
	case MessageTypeTask:
		return participant.view.MergeTask(datura.As[a2a.Task](artifact))
	case MessageTypeTaskClaim:
		return participant.view.MergeTaskClaim(datura.As[TaskClaim](artifact))
	case MessageTypeTaskStatus:
		return participant.view.MergeTaskStatus(datura.As[a2a.TaskStatusUpdateEvent](artifact))
	case MessageTypeSignal:
		return participant.view.MergeSignal(datura.As[Signal](artifact))
	case MessageTypeMetric:
		return participant.view.MergeMetric(datura.As[Metric](artifact))
	case MessageTypeContention:
		return participant.view.MergeContention(datura.As[Contention](artifact))
	default:
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm artifact type %q is unsupported", qpool.BusMessageType(artifact)),
			nil,
		)
	}
}

/*
Drain ingests all pending mesh rumors without blocking.
*/
func (participant *Participant) Drain() error {
	for {
		artifact := participant.subscriber.Poll()

		if artifact == nil {
			return nil
		}

		if err := participant.ReceiveArtifact(artifact); err != nil {
			return err
		}
	}
}

/*
Announce publishes a roadmap or situational payload to the mesh.
*/
func (participant *Participant) Announce(topic, payload string) error {
	rumor := NewRumorAt(
		KindAnnounce,
		participant.actorID,
		participant.actorName,
		participant.role,
		time.Now(),
	)

	rumor.Topic = topic
	rumor.Payload = payload

	if err := participant.view.Merge(rumor); err != nil {
		return err
	}

	return participant.mesh.Publish(participant.actorID, rumor)
}

/*
TryClaim acquires a filesystem lease then publishes a gossip claim.
*/
func (participant *Participant) TryClaim(prefix string) error {
	if err := participant.coordinator.AcquireID(prefix, participant.actorID); err != nil {
		return participant.contentionError(prefix, err)
	}

	return participant.publishClaim(prefix)
}

/*
Renew touches a held lease and republishes its claim rumor.
*/
func (participant *Participant) Renew(prefix string) error {
	if err := participant.coordinator.TouchID(prefix, participant.actorID); err != nil {
		return err
	}

	return participant.publishClaim(prefix)
}

func (participant *Participant) publishClaim(prefix string) error {
	rumor := NewRumorAt(
		KindClaim,
		participant.actorID,
		participant.actorName,
		participant.role,
		time.Now(),
	)

	rumor.Prefix = prefix

	if err := participant.view.Merge(rumor); err != nil {
		return err
	}

	return participant.mesh.Publish(participant.actorID, rumor)
}

/*
Release drops a lease and publishes a gossip release rumor.
*/
func (participant *Participant) Release(prefix string) error {
	if err := participant.coordinator.ReleaseID(
		prefix,
		participant.actorID,
	); err != nil {
		return err
	}

	rumor := NewRumorAt(
		KindRelease,
		participant.actorID,
		participant.actorName,
		participant.role,
		time.Now(),
	)

	rumor.Prefix = prefix

	if err := participant.view.Merge(rumor); err != nil {
		return err
	}

	return participant.mesh.Publish(participant.actorID, rumor)
}

/*
PublishStatus emits a heartbeat-style status rumor.
*/
func (participant *Participant) PublishStatus(state string) error {
	rumor := NewRumorAt(
		KindStatus,
		participant.actorID,
		participant.actorName,
		participant.role,
		time.Now(),
	)

	rumor.State = state

	if err := participant.view.Merge(rumor); err != nil {
		return err
	}

	return participant.mesh.Publish(participant.actorID, rumor)
}
