package conversation

import (
	"fmt"
	"maps"

	"github.com/theapemachine/animal/internal"
	"github.com/theapemachine/animal/swarm"
)

type salonSnapshot struct {
	turns   []Turn
	stances map[string][][]string
	seen    map[string]struct{}
}

func newSalonSnapshot() salonSnapshot {
	return salonSnapshot{
		turns:   make([]Turn, 0),
		stances: make(map[string][][]string),
		seen:    make(map[string]struct{}),
	}
}

func cloneSalonSnapshot(snapshot salonSnapshot) salonSnapshot {
	turns := append([]Turn(nil), snapshot.turns...)
	stances := make(map[string][][]string, len(snapshot.stances))

	for actorID, history := range snapshot.stances {
		copied := make([][]string, len(history))
		copy(copied, history)
		stances[actorID] = copied
	}

	seen := make(map[string]struct{}, len(snapshot.seen))
	maps.Copy(seen, snapshot.seen)

	return salonSnapshot{
		turns:   turns,
		stances: stances,
		seen:    seen,
	}
}

/*
SalonRegistry merges gossip into a shared transcript and stance history.
*/
type SalonRegistry struct {
	state *internal.Snapshot[salonSnapshot]
}

/*
NewSalonRegistry instantiates an empty salon registry.
*/
func NewSalonRegistry() *SalonRegistry {
	return &SalonRegistry{
		state: internal.NewSnapshot(newSalonSnapshot()),
	}
}

/*
Apply ingests one announce record idempotently.
*/
func (registry *SalonRegistry) Apply(record swarm.AnnounceRecord) error {
	key := fmt.Sprintf("%d:%s:%s:%s", record.At, record.ActorID, record.Topic, record.Payload)
	var applyErr error

	registry.state.Update(func(snapshot salonSnapshot) salonSnapshot {
		if _, ok := snapshot.seen[key]; ok {
			return snapshot
		}

		updated := cloneSalonSnapshot(snapshot)
		updated.seen[key] = struct{}{}

		switch record.Topic {
		case TopicTurn:
			turn, err := ParseTurn(record)
			if err != nil {
				applyErr = err
				return snapshot
			}

			updated.turns = append(updated.turns, turn)
		case TopicStance:
			themes, err := ParseStance(record)
			if err != nil {
				applyErr = err
				return snapshot
			}

			recordStance(&updated, record.ActorID, themes)
		default:
			return snapshot
		}

		return updated
	})

	return applyErr
}

/*
Turns returns a snapshot of merged transcript lines.
*/
func (registry *SalonRegistry) Turns() []Turn {
	turns := registry.state.Load().turns
	out := make([]Turn, len(turns))
	copy(out, turns)

	return out
}

/*
ThemeSet returns the merged theme tags recorded for one actor.
*/
func (registry *SalonRegistry) ThemeSet(actorID string) map[string]struct{} {
	return mergeThemeSets(registry.state.Load().stances[actorID])
}

func recordStance(snapshot *salonSnapshot, actorID string, themes []string) {
	history := snapshot.stances[actorID]
	history = append(history, themes)

	if len(history) > 4 {
		history = history[len(history)-4:]
	}

	snapshot.stances[actorID] = history
}

func mergeThemeSets(history [][]string) map[string]struct{} {
	themes := make(map[string]struct{})

	for _, batch := range history {
		for _, theme := range batch {
			themes[theme] = struct{}{}
		}
	}

	return themes
}
