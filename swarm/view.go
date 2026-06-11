package swarm

import (
	"fmt"
	"sync"
	"time"
)

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

type announceRecord struct {
	actorID   string
	actorName string
	role      string
	topic     string
	payload   string
	at        int64
}

type statusRecord struct {
	actorName string
	role      string
	state     string
	at        int64
}

/*
View holds the merged situational picture from gossip rumors.
*/
type View struct {
	mu         sync.RWMutex
	gossipTTL  time.Duration
	claims     map[string]claimRecord
	announces  []announceRecord
	statuses map[string]statusRecord
}

/*
NewView instantiates an empty local view with gossip expiration.
*/
func NewView(gossipTTL time.Duration) (*View, error) {
	if gossipTTL <= 0 {
		return nil, fmt.Errorf("swarm: gossip TTL is required")
	}

	return &View{
		gossipTTL:  gossipTTL,
		claims:     make(map[string]claimRecord),
		announces:  make([]announceRecord, 0),
		statuses: make(map[string]statusRecord),
	}, nil
}

/*
Merge applies rumor into the local view using last-write-wins ordering.
*/
func (view *View) Merge(rumor Rumor) error {
	if err := rumor.Validate(); err != nil {
		return err
	}

	view.mu.Lock()
	defer view.mu.Unlock()

	switch rumor.Kind {
	case KindClaim:
		view.mergeClaim(rumor)
	case KindRelease:
		view.mergeRelease(rumor)
	case KindAnnounce:
		view.mergeAnnounce(rumor)
	case KindStatus:
		view.mergeStatus(rumor)
	}

	return nil
}

/*
PurgeExpired drops gossip entries older than the configured TTL.
*/
func (view *View) PurgeExpired(now time.Time) {
	cutoff := now.Add(-view.gossipTTL).UnixNano()

	view.mu.Lock()
	defer view.mu.Unlock()

	for prefix, record := range view.claims {
		if record.at >= cutoff {
			continue
		}

		delete(view.claims, prefix)
	}

	filtered := view.announces[:0]

	for _, record := range view.announces {
		if record.at >= cutoff {
			filtered = append(filtered, record)
		}
	}

	view.announces = filtered

	for actorID, record := range view.statuses {
		if record.at >= cutoff {
			continue
		}

		delete(view.statuses, actorID)
	}
}

/*
ClaimHolder returns the actor currently claiming prefix in gossip state.
*/
func (view *View) ClaimHolder(prefix string) (string, bool) {
	view.mu.RLock()
	defer view.mu.RUnlock()

	record, ok := view.claims[prefix]

	if !ok {
		return "", false
	}

	return record.actorID, true
}

/*
IsPrefixFree reports whether gossip shows no active claim on prefix.
*/
func (view *View) IsPrefixFree(prefix string) bool {
	view.mu.RLock()
	defer view.mu.RUnlock()

	_, ok := view.claims[prefix]

	return !ok
}

/*
RecentAnnounces returns a snapshot of non-expired announcements.
*/
func (view *View) RecentAnnounces() []AnnounceRecord {
	view.mu.RLock()
	defer view.mu.RUnlock()

	out := make([]AnnounceRecord, 0, len(view.announces))

	for _, record := range view.announces {
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

func (view *View) mergeClaim(rumor Rumor) {
	record, ok := view.claims[rumor.Prefix]

	if ok && record.at > rumor.At {
		return
	}

	view.claims[rumor.Prefix] = claimRecord{
		actorID:   rumor.ActorID,
		actorName: rumor.ActorName,
		role:      rumor.Role,
		at:        rumor.At,
	}
}

func (view *View) mergeRelease(rumor Rumor) {
	record, ok := view.claims[rumor.Prefix]

	if !ok {
		return
	}

	if record.actorID != rumor.ActorID {
		return
	}

	if record.at > rumor.At {
		return
	}

	delete(view.claims, rumor.Prefix)
}

func (view *View) mergeAnnounce(rumor Rumor) {
	view.announces = append(view.announces, announceRecord{
		actorID:   rumor.ActorID,
		actorName: rumor.ActorName,
		role:      rumor.Role,
		topic:     rumor.Topic,
		payload:   rumor.Payload,
		at:        rumor.At,
	})
}

func (view *View) mergeStatus(rumor Rumor) {
	record, ok := view.statuses[rumor.ActorID]

	if ok && record.at > rumor.At {
		return
	}

	view.statuses[rumor.ActorID] = statusRecord{
		actorName: rumor.ActorName,
		role:      rumor.Role,
		state:     rumor.State,
		at:        rumor.At,
	}
}
