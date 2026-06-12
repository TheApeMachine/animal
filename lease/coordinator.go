package lease

import (
	"fmt"
	"maps"
	"time"

	"github.com/theapemachine/animal/actor"
	"github.com/theapemachine/animal/internal"
)

type coordinatorSnapshot struct {
	leases map[string]leaseRecord
}

func newCoordinatorSnapshot() coordinatorSnapshot {
	return coordinatorSnapshot{
		leases: make(map[string]leaseRecord),
	}
}

func cloneCoordinatorSnapshot(snapshot coordinatorSnapshot) coordinatorSnapshot {
	leases := make(map[string]leaseRecord, len(snapshot.leases))
	maps.Copy(leases, snapshot.leases)

	return coordinatorSnapshot{leases: leases}
}

/*
Coordinator tracks exclusive prefix leases over a KeySpace.
Leases expire when idle longer than Options.IdleTTL.
*/
type Coordinator struct {
	state    *internal.Snapshot[coordinatorSnapshot]
	keySpace KeySpace
	idleTTL  time.Duration
	now      func() time.Time
}

/*
NewCoordinator instantiates an in-process lease registry.
*/
func NewCoordinator(options Options) (*Coordinator, error) {
	if options.KeySpace == nil {
		return nil, fmt.Errorf("lease: key space is required")
	}

	if options.IdleTTL <= 0 {
		return nil, fmt.Errorf("lease: idle TTL is required")
	}

	return &Coordinator{
		state:    internal.NewSnapshot(newCoordinatorSnapshot()),
		keySpace: options.KeySpace,
		idleTTL:  options.IdleTTL,
		now:      time.Now,
	}, nil
}

/*
Acquire grants identity exclusive access to prefix.
*/
func (coordinator *Coordinator) Acquire(prefix string, identity *actor.Identity) error {
	if identity == nil {
		return fmt.Errorf("lease: identity is required")
	}

	return coordinator.AcquireID(prefix, identity.ID)
}

/*
AcquireID grants actorID exclusive access to prefix without an Identity value.
*/
func (coordinator *Coordinator) AcquireID(prefix, actorID string) error {
	if actorID == "" {
		return fmt.Errorf("lease: actor ID is required")
	}

	leaseKey, err := coordinator.keySpace.Normalize(prefix)
	if err != nil {
		return err
	}

	var acquireErr error

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)
		coordinator.purgeExpiredSnapshot(&updated, coordinator.now())

		record, ok := updated.leases[leaseKey]
		if ok && record.actorID != actorID {
			acquireErr = fmt.Errorf("lease: key %q held by actor %q", prefix, record.actorID)
			return snapshot
		}

		updated.leases[leaseKey] = leaseRecord{
			actorID:  actorID,
			lastUsed: coordinator.now(),
		}

		return updated
	})

	return acquireErr
}

/*
Release drops prefix when held by identity.
*/
func (coordinator *Coordinator) Release(prefix string, identity *actor.Identity) error {
	if identity == nil {
		return fmt.Errorf("lease: identity is required")
	}

	return coordinator.ReleaseID(prefix, identity.ID)
}

/*
ReleaseID drops prefix when held by actorID.
*/
func (coordinator *Coordinator) ReleaseID(prefix, actorID string) error {
	if actorID == "" {
		return fmt.Errorf("lease: actor ID is required")
	}

	leaseKey, err := coordinator.keySpace.Normalize(prefix)
	if err != nil {
		return err
	}

	var releaseErr error

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)
		coordinator.purgeExpiredSnapshot(&updated, coordinator.now())

		record, ok := updated.leases[leaseKey]
		if !ok {
			releaseErr = fmt.Errorf("lease: key %q is not leased", prefix)
			return snapshot
		}

		if record.actorID != actorID {
			releaseErr = fmt.Errorf("lease: key %q held by actor %q", prefix, record.actorID)
			return snapshot
		}

		delete(updated.leases, leaseKey)

		return updated
	})

	return releaseErr
}

/*
TouchID renews the idle timer for a lease held by actorID.
*/
func (coordinator *Coordinator) TouchID(prefix, actorID string) error {
	if actorID == "" {
		return fmt.Errorf("lease: actor ID is required")
	}

	leaseKey, err := coordinator.keySpace.Normalize(prefix)
	if err != nil {
		return err
	}

	var touchErr error

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)
		coordinator.purgeExpiredSnapshot(&updated, coordinator.now())

		record, ok := updated.leases[leaseKey]
		if !ok {
			touchErr = fmt.Errorf("lease: key %q is not leased", prefix)
			return snapshot
		}

		if record.actorID != actorID {
			touchErr = fmt.Errorf("lease: key %q held by actor %q", prefix, record.actorID)
			return snapshot
		}

		record.lastUsed = coordinator.now()
		updated.leases[leaseKey] = record

		return updated
	})

	return touchErr
}

/*
ObserveRead reports when key is under another actor's active lease.

A ChangingError is advisory: the caller should retry later or continue without
depending on the resource's current shape.
*/
func (coordinator *Coordinator) ObserveRead(key string, principal Principal) error {
	normalized, err := coordinator.keySpace.Normalize(key)
	if err != nil {
		return err
	}

	var observeErr error

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)
		coordinator.purgeExpiredSnapshot(&updated, coordinator.now())

		leaseKey, holder := coordinator.leaseHolderFromSnapshot(updated, normalized)
		if holder == "" || holder == principal.ActorID {
			return updated
		}

		observeErr = &ChangingError{
			Key:      key,
			LeaseKey: leaseKey,
			ActorID:  holder,
		}

		return updated
	})

	return observeErr
}

/*
CanWrite checks read-only mode, allowed prefixes, and active lease requirements.
*/
func (coordinator *Coordinator) CanWrite(key string, principal Principal) error {
	if principal.ReadOnly {
		return fmt.Errorf("lease: actor %q is read-only", principal.ActorID)
	}

	normalized, err := coordinator.keySpace.Normalize(key)
	if err != nil {
		return err
	}

	if len(principal.AllowedPrefixes) > 0 && !coordinator.matchesAnyPrefix(normalized, principal.AllowedPrefixes) {
		return fmt.Errorf("lease: actor %q cannot write outside allowed prefixes", principal.ActorID)
	}

	if !principal.RequireLease {
		coordinator.touchCoveringLease(normalized, principal.ActorID)
		return nil
	}

	var writeErr error

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)
		coordinator.purgeExpiredSnapshot(&updated, coordinator.now())

		for leaseKey, record := range updated.leases {
			if !coordinator.keySpace.Covers(leaseKey, normalized) {
				continue
			}

			if record.actorID != principal.ActorID {
				continue
			}

			record.lastUsed = coordinator.now()
			updated.leases[leaseKey] = record

			return updated
		}

		writeErr = fmt.Errorf("lease: actor %q lacks an active lease for %q", principal.ActorID, key)

		return snapshot
	})

	return writeErr
}

func (coordinator *Coordinator) touchCoveringLease(normalizedKey, actorID string) {
	if actorID == "" {
		return
	}

	coordinator.state.Update(func(snapshot coordinatorSnapshot) coordinatorSnapshot {
		updated := cloneCoordinatorSnapshot(snapshot)
		coordinator.purgeExpiredSnapshot(&updated, coordinator.now())

		for leaseKey, record := range updated.leases {
			if record.actorID != actorID {
				continue
			}

			if !coordinator.keySpace.Covers(leaseKey, normalizedKey) {
				continue
			}

			record.lastUsed = coordinator.now()
			updated.leases[leaseKey] = record

			return updated
		}

		return snapshot
	})
}

func (coordinator *Coordinator) purgeExpiredSnapshot(snapshot *coordinatorSnapshot, now time.Time) {
	for leaseKey, record := range snapshot.leases {
		if now.Sub(record.lastUsed) <= coordinator.idleTTL {
			continue
		}

		delete(snapshot.leases, leaseKey)
	}
}

func (coordinator *Coordinator) leaseHolderFromSnapshot(
	snapshot coordinatorSnapshot, normalizedKey string,
) (string, string) {
	var bestLeaseKey string
	var holder string

	for leaseKey, record := range snapshot.leases {
		if !coordinator.keySpace.Covers(leaseKey, normalizedKey) {
			continue
		}

		if len(leaseKey) <= len(bestLeaseKey) {
			continue
		}

		bestLeaseKey = leaseKey
		holder = record.actorID
	}

	return bestLeaseKey, holder
}

func (coordinator *Coordinator) matchesAnyPrefix(normalizedKey string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if coordinator.keySpace.Covers(prefix, normalizedKey) {
			return true
		}
	}

	return false
}
