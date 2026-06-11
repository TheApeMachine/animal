package ownership

import (
	"fmt"

	"github.com/theapemachine/animal/actor"
	"github.com/theapemachine/animal/lease"
)

/*
Leasable is a resource that can be exclusively owned by one actor at a time.
*/
type Leasable interface {
	Acquire(*actor.Identity) error
	Release() error
}

/*
Resource binds a lease key to the shared coordinator.
*/
type Resource struct {
	coordinator *lease.Coordinator
	key         string
	holder      *actor.Identity
}

/*
NewResource registers key with coordinator for Leasable operations.
*/
func NewResource(coordinator *lease.Coordinator, key string) (*Resource, error) {
	if coordinator == nil {
		return nil, fmt.Errorf("ownership: lease coordinator is required")
	}

	if key == "" {
		return nil, fmt.Errorf("ownership: lease key is required")
	}

	return &Resource{
		coordinator: coordinator,
		key:         key,
	}, nil
}

/*
Acquire grants identity exclusive access to the resource key.
*/
func (resource *Resource) Acquire(identity *actor.Identity) error {
	if err := resource.coordinator.Acquire(resource.key, identity); err != nil {
		return &LeasableError{Identity: identity, Err: err}
	}

	resource.holder = identity
	return nil
}

/*
Release drops the lease when held by the current holder.
*/
func (resource *Resource) Release() error {
	if resource.holder == nil {
		return fmt.Errorf("ownership: resource %q is not held", resource.key)
	}

	if err := resource.coordinator.Release(resource.key, resource.holder); err != nil {
		return &LeasableError{Identity: resource.holder, Err: err}
	}

	resource.holder = nil
	return nil
}

/*
LeasableError wraps lease failures with the acting identity.
*/
type LeasableError struct {
	Identity *actor.Identity
	Err      error
}

func (leasableErr *LeasableError) Error() string {
	if leasableErr.Identity == nil {
		return fmt.Sprintf("ownership: %s", leasableErr.Err.Error())
	}

	return fmt.Sprintf("ownership: %s: %s", leasableErr.Identity.ID, leasableErr.Err.Error())
}

func (leasableErr *LeasableError) Unwrap() error {
	return leasableErr.Err
}
