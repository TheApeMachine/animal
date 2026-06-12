package swarm

import (
	"fmt"
	"maps"
	"time"

	"github.com/theapemachine/animal/internal"
)

/*
claimRecord stores the latest gossip claim for one lease prefix inside a local View.
It is unexported because callers consume merged state through AnnounceRecord and ClaimHolder.
*/
type claimRecord struct {
	actorID   string
	actorName string
	role      string
	at        int64
}

/*
AnnounceRecord is one merged announcement visible in the local view.
*/
type AnnounceRecord struct {
	ActorID   string
	ActorName string
	Role      string
	Topic     string
	Payload   string
	At        int64
}

/*
announceRecord is the internal storage shape for one gossip announcement before export.
Keeping actor metadata denormalized here avoids repeated Rumor parsing during view merges.
*/
type announceRecord struct {
	actorID   string
	actorName string
	role      string
	topic     string
	payload   string
	at        int64
}

/*
statusRecord tracks the most recent heartbeat-style state broadcast from one actor.
Last-write-wins ordering matches how other gossip kinds are merged into the View.
*/
type statusRecord struct {
	actorName string
	role      string
	state     string
	at        int64
}

type viewSnapshot struct {
	claims    map[string]claimRecord
	announces []announceRecord
	statuses  map[string]statusRecord
}

func newViewSnapshot() viewSnapshot {
	return viewSnapshot{
		claims:    make(map[string]claimRecord),
		announces: make([]announceRecord, 0),
		statuses:  make(map[string]statusRecord),
	}
}

func cloneViewSnapshot(snapshot viewSnapshot) viewSnapshot {
	claims := make(map[string]claimRecord, len(snapshot.claims))
	maps.Copy(claims, snapshot.claims)

	statuses := make(map[string]statusRecord, len(snapshot.statuses))
	maps.Copy(statuses, snapshot.statuses)

	announces := append([]announceRecord(nil), snapshot.announces...)

	return viewSnapshot{
		claims:    claims,
		announces: announces,
		statuses:  statuses,
	}
}

/*
View holds the merged situational picture from gossip rumors.
*/
type View struct {
	state     *internal.Snapshot[viewSnapshot]
	gossipTTL time.Duration
}

/*
NewView instantiates an empty local view with gossip expiration.
*/
func NewView(gossipTTL time.Duration) (*View, error) {
	if gossipTTL <= 0 {
		return nil, fmt.Errorf("swarm: gossip TTL is required")
	}

	return &View{
		state:     internal.NewSnapshot(newViewSnapshot()),
		gossipTTL: gossipTTL,
	}, nil
}

/*
Merge applies rumor into the local view using last-write-wins ordering.
*/
func (view *View) Merge(rumor Rumor) error {
	if err := rumor.Validate(); err != nil {
		return err
	}

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)

		switch rumor.Kind {
		case KindClaim:
			view.mergeClaim(&updated, rumor)
		case KindRelease:
			view.mergeRelease(&updated, rumor)
		case KindAnnounce:
			view.mergeAnnounce(&updated, rumor)
		case KindStatus:
			view.mergeStatus(&updated, rumor)
		}

		return updated
	})

	return nil
}

/*
PurgeExpired drops gossip entries older than the configured TTL.
*/
func (view *View) PurgeExpired(now time.Time) {
	cutoff := now.Add(-view.gossipTTL).UnixNano()

	view.state.Update(func(snapshot viewSnapshot) viewSnapshot {
		updated := cloneViewSnapshot(snapshot)

		for prefix, record := range updated.claims {
			if record.at >= cutoff {
				continue
			}

			delete(updated.claims, prefix)
		}

		filtered := updated.announces[:0]

		for _, record := range updated.announces {
			if record.at >= cutoff {
				filtered = append(filtered, record)
			}
		}

		updated.announces = filtered

		for actorID, record := range updated.statuses {
			if record.at >= cutoff {
				continue
			}

			delete(updated.statuses, actorID)
		}

		return updated
	})
}

/*
ClaimHolder returns the actor currently claiming prefix in gossip state.
*/
func (view *View) ClaimHolder(prefix string) (string, bool) {
	record, ok := view.state.Load().claims[prefix]

	if !ok {
		return "", false
	}

	return record.actorID, true
}

/*
IsPrefixFree reports whether gossip shows no active claim on prefix.
*/
func (view *View) IsPrefixFree(prefix string) bool {
	_, ok := view.state.Load().claims[prefix]

	return !ok
}

/*
RecentAnnounces returns a snapshot of non-expired announcements.
*/
func (view *View) RecentAnnounces() []AnnounceRecord {
	snapshot := view.state.Load()
	out := make([]AnnounceRecord, 0, len(snapshot.announces))

	for _, record := range snapshot.announces {
		out = append(out, AnnounceRecord{
			ActorID:   record.actorID,
			ActorName: record.actorName,
			Role:      record.role,
			Topic:     record.topic,
			Payload:   record.payload,
			At:        record.at,
		})
	}

	return out
}

func (view *View) mergeClaim(snapshot *viewSnapshot, rumor Rumor) {
	record, ok := snapshot.claims[rumor.Prefix]

	if ok && record.at > rumor.At {
		return
	}

	snapshot.claims[rumor.Prefix] = claimRecord{
		actorID:   rumor.ActorID,
		actorName: rumor.ActorName,
		role:      rumor.Role,
		at:        rumor.At,
	}
}

func (view *View) mergeRelease(snapshot *viewSnapshot, rumor Rumor) {
	record, ok := snapshot.claims[rumor.Prefix]

	if !ok {
		return
	}

	if record.actorID != rumor.ActorID {
		return
	}

	if record.at > rumor.At {
		return
	}

	delete(snapshot.claims, rumor.Prefix)
}

func (view *View) mergeAnnounce(snapshot *viewSnapshot, rumor Rumor) {
	snapshot.announces = append(snapshot.announces, announceRecord{
		actorID:   rumor.ActorID,
		actorName: rumor.ActorName,
		role:      rumor.Role,
		topic:     rumor.Topic,
		payload:   rumor.Payload,
		at:        rumor.At,
	})
}

func (view *View) mergeStatus(snapshot *viewSnapshot, rumor Rumor) {
	record, ok := snapshot.statuses[rumor.ActorID]

	if ok && record.at > rumor.At {
		return
	}

	snapshot.statuses[rumor.ActorID] = statusRecord{
		actorName: rumor.ActorName,
		role:      rumor.Role,
		state:     rumor.State,
		at:        rumor.At,
	}
}
