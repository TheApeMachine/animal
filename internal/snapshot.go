package internal

import "sync/atomic"

/*
Snapshot stores an immutable value updated with copy-on-write CAS loops.
Readers load atomically; writers clone, mutate, and retry on conflict.
*/
type Snapshot[T any] struct {
	value atomic.Pointer[T]
}

/*
NewSnapshot stores the initial value for lock-free reads and CAS updates.
*/
func NewSnapshot[T any](initial T) *Snapshot[T] {
	snapshot := &Snapshot[T]{}
	initialValue := new(T)
	*initialValue = initial
	snapshot.value.Store(initialValue)

	return snapshot
}

/*
Load returns the current immutable snapshot value.
*/
func (snapshot *Snapshot[T]) Load() T {
	return *snapshot.value.Load()
}

/*
Update applies change to a clone and publishes it when the CAS succeeds.
*/
func (snapshot *Snapshot[T]) Update(change func(T) T) {
	for {
		current := snapshot.value.Load()
		updated := change(*current)
		next := new(T)
		*next = updated

		if snapshot.value.CompareAndSwap(current, next) {
			return
		}
	}
}
