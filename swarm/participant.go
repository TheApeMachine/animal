package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/theapemachine/animal/lease"
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
func (participant *Participant) Incoming() (*qpool.QValue[any], error) {
	return participant.subscriber.Wait(participant.ctx)
}

/*
Poll returns the next pending mesh rumor without blocking.
*/
func (participant *Participant) Poll() *qpool.QValue[any] {
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
Drain ingests all pending mesh rumors without blocking.
*/
func (participant *Participant) Drain() error {
	for {
		qv := participant.subscriber.Poll()
		if qv == nil {
			return nil
		}

		rumor, ok := qv.Value.(Rumor)

		if !ok {
			return fmt.Errorf("swarm: mesh received non-rumor payload for actor %q", participant.actorID)
		}

		if err := participant.Receive(rumor); err != nil {
			return err
		}
	}
}

/*
Announce publishes a roadmap or situational payload to the mesh.
*/
func (participant *Participant) Announce(topic, payload string) error {
	rumor := NewRumorAt(KindAnnounce, participant.actorID, participant.actorName, participant.role, time.Now())
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
		return err
	}

	rumor := NewRumorAt(KindClaim, participant.actorID, participant.actorName, participant.role, time.Now())
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
	if err := participant.coordinator.ReleaseID(prefix, participant.actorID); err != nil {
		return err
	}

	rumor := NewRumorAt(KindRelease, participant.actorID, participant.actorName, participant.role, time.Now())
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
	rumor := NewRumorAt(KindStatus, participant.actorID, participant.actorName, participant.role, time.Now())
	rumor.State = state

	if err := participant.view.Merge(rumor); err != nil {
		return err
	}

	return participant.mesh.Publish(participant.actorID, rumor)
}

/*
TryClaimConfigured attempts claims on configured prefixes that gossip shows as free.
*/
func (participant *Participant) TryClaimConfigured() (string, error) {
	participant.view.PurgeExpired(time.Now())

	for _, prefix := range participant.claimPrefixes {
		if !participant.view.IsPrefixFree(prefix) {
			continue
		}

		if err := participant.TryClaim(prefix); err != nil {
			continue
		}

		return prefix, nil
	}

	return "", fmt.Errorf("swarm: no configured prefix available for actor %q", participant.actorID)
}
