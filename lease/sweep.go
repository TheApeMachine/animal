package lease

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

/*
ExpiredLease describes one lease removed by a coordinator sweep.
It carries the original prefix string so gossip releases match the claim peers already merged.
*/
type ExpiredLease struct {
	ActorID  string
	Prefix   string
	LastUsed time.Time
}

/*
SweepExpired removes idle leases and returns the records that were reclaimed.
*/
func (coordinator *Coordinator) SweepExpired(now time.Time) ([]ExpiredLease, error) {
	if now.IsZero() {
		return nil, fmt.Errorf("lease: sweep time is required")
	}

	expired := make([]ExpiredLease, 0)

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)

		for leaseKey, record := range updated.leases {
			if now.Sub(record.lastUsed) <= coordinator.idleTTL {
				continue
			}

			expired = append(expired, ExpiredLease{
				ActorID:  record.actorID,
				Prefix:   record.prefix,
				LastUsed: record.lastUsed,
			})

			delete(updated.leases, leaseKey)
		}

		return updated
	})

	slices.SortFunc(expired, func(firstLease, secondLease ExpiredLease) int {
		if diff := cmp.Compare(firstLease.Prefix, secondLease.Prefix); diff != 0 {
			return diff
		}

		return cmp.Compare(firstLease.ActorID, secondLease.ActorID)
	})

	return expired, nil
}
