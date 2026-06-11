package lease

import (
	"fmt"
	"sync"
	"time"

	"github.com/theapemachine/animal/actor"
)

/*
Coordinator tracks exclusive prefix leases over a KeySpace.
Leases expire when idle longer than Options.IdleTTL.
*/
type Coordinator struct {
	mu       sync.RWMutex
	keySpace KeySpace
	idleTTL  time.Duration
	now      func() time.Time
	leases   map[string]leaseRecord
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
		keySpace: options.KeySpace,
		idleTTL:  options.IdleTTL,
		now:      time.Now,
		leases:   make(map[string]leaseRecord),
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

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	coordinator.purgeExpiredLocked(coordinator.now())

	record, ok := coordinator.leases[leaseKey]
	if ok && record.actorID != actorID {
		return fmt.Errorf("lease: key %q held by actor %q", prefix, record.actorID)
	}

	coordinator.leases[leaseKey] = leaseRecord{
		actorID:  actorID,
		lastUsed: coordinator.now(),
	}

	return nil
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

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	coordinator.purgeExpiredLocked(coordinator.now())

	record, ok := coordinator.leases[leaseKey]
	if !ok {
		return fmt.Errorf("lease: key %q is not leased", prefix)
	}

	if record.actorID != actorID {
		return fmt.Errorf("lease: key %q held by actor %q", prefix, record.actorID)
	}

	delete(coordinator.leases, leaseKey)
	return nil
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

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	coordinator.purgeExpiredLocked(coordinator.now())

	record, ok := coordinator.leases[leaseKey]
	if !ok {
		return fmt.Errorf("lease: key %q is not leased", prefix)
	}

	if record.actorID != actorID {
		return fmt.Errorf("lease: key %q held by actor %q", prefix, record.actorID)
	}

	record.lastUsed = coordinator.now()
	coordinator.leases[leaseKey] = record
	return nil
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

	coordinator.mu.Lock()
	coordinator.purgeExpiredLocked(coordinator.now())
	coordinator.mu.Unlock()

	coordinator.mu.RLock()
	defer coordinator.mu.RUnlock()

	leaseKey, holder := coordinator.leaseHolder(normalized)
	if holder == "" || holder == principal.ActorID {
		return nil
	}

	return &ChangingError{
		Key:      key,
		LeaseKey: leaseKey,
		ActorID:  holder,
	}
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

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	coordinator.purgeExpiredLocked(coordinator.now())

	for leaseKey, record := range coordinator.leases {
		if !coordinator.keySpace.Covers(leaseKey, normalized) {
			continue
		}

		if record.actorID != principal.ActorID {
			continue
		}

		record.lastUsed = coordinator.now()
		coordinator.leases[leaseKey] = record
		return nil
	}

	return fmt.Errorf("lease: actor %q lacks an active lease for %q", principal.ActorID, key)
}

func (coordinator *Coordinator) touchCoveringLease(normalizedKey, actorID string) {
	if actorID == "" {
		return
	}

	coordinator.mu.Lock()
	defer coordinator.mu.Unlock()

	coordinator.purgeExpiredLocked(coordinator.now())

	for leaseKey, record := range coordinator.leases {
		if record.actorID != actorID {
			continue
		}

		if !coordinator.keySpace.Covers(leaseKey, normalizedKey) {
			continue
		}

		record.lastUsed = coordinator.now()
		coordinator.leases[leaseKey] = record
		return
	}
}

func (coordinator *Coordinator) purgeExpiredLocked(now time.Time) {
	for leaseKey, record := range coordinator.leases {
		if now.Sub(record.lastUsed) <= coordinator.idleTTL {
			continue
		}

		delete(coordinator.leases, leaseKey)
	}
}

func (coordinator *Coordinator) leaseHolder(normalizedKey string) (string, string) {
	var bestLeaseKey string
	var holder string

	for leaseKey, record := range coordinator.leases {
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
