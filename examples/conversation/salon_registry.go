package conversation

import (
	"fmt"
	"sync"

	"github.com/theapemachine/animal/swarm"
)

/*
SalonRegistry merges gossip into a shared transcript and stance history.
*/
type SalonRegistry struct {
	mu      sync.RWMutex
	turns   []Turn
	stances map[string][][]string
	seen    map[string]struct{}
}

/*
NewSalonRegistry instantiates an empty salon registry.
*/
func NewSalonRegistry() *SalonRegistry {
	return &SalonRegistry{
		turns:   make([]Turn, 0),
		stances: make(map[string][][]string),
		seen:    make(map[string]struct{}),
	}
}

/*
Apply ingests one announce record idempotently.
*/
func (registry *SalonRegistry) Apply(record swarm.AnnounceRecord) error {
	key := fmt.Sprintf("%d:%s:%s:%s", record.At, record.ActorID, record.Topic, record.Payload)

	registry.mu.Lock()
	defer registry.mu.Unlock()

	if _, ok := registry.seen[key]; ok {
		return nil
	}

	registry.seen[key] = struct{}{}

	switch record.Topic {
	case TopicTurn:
		turn, err := ParseTurn(record)
		if err != nil {
			return err
		}

		registry.turns = append(registry.turns, turn)
	case TopicStance:
		themes, err := ParseStance(record)
		if err != nil {
			return err
		}

		registry.recordStanceLocked(record.ActorID, themes)
	default:
		return nil
	}

	return nil
}

/*
Turns returns a snapshot of merged transcript lines.
*/
func (registry *SalonRegistry) Turns() []Turn {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	out := make([]Turn, len(registry.turns))
	copy(out, registry.turns)

	return out
}

/*
ThemeSet returns the merged theme tags recorded for one actor.
*/
func (registry *SalonRegistry) ThemeSet(actorID string) map[string]struct{} {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	return mergeThemeSets(registry.stances[actorID])
}

func (registry *SalonRegistry) recordStanceLocked(actorID string, themes []string) {
	history := registry.stances[actorID]
	history = append(history, themes)

	if len(history) > 4 {
		history = history[len(history)-4:]
	}

	registry.stances[actorID] = history
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
