package lease

import (
	"errors"
	"fmt"
)

/*
ConflictError reports that another actor already holds an overlapping lease.
*/
type ConflictError struct {
	Key      string
	LeaseKey string
	ActorID  string
}

func (conflict *ConflictError) Error() string {
	return fmt.Sprintf(
		"lease: key %q held by actor %q under lease %q",
		conflict.Key,
		conflict.ActorID,
		conflict.LeaseKey,
	)
}

/*
AsConflict reports whether err is a lease conflict.
*/
func AsConflict(err error) (*ConflictError, bool) {
	var conflict *ConflictError

	if !errors.As(err, &conflict) {
		return nil, false
	}

	return conflict, true
}

/*
ChangingError is an advisory signal, not a hard read failure.

Another actor holds a lease covering key, so the resource shape may change before
a stable read is possible.
*/
type ChangingError struct {
	Key      string
	LeaseKey string
	ActorID  string
}

func (changing *ChangingError) Error() string {
	return fmt.Sprintf(
		"resource %q is being changed by actor %q under lease %q: "+
			"retry after the lease is released, or continue work that does not depend on this resource's current shape",
		changing.Key,
		changing.ActorID,
		changing.LeaseKey,
	)
}

/*
AsChanging reports whether err is an advisory changing-resource signal.
*/
func AsChanging(err error) (*ChangingError, bool) {
	var changing *ChangingError

	if !errors.As(err, &changing) {
		return nil, false
	}

	return changing, true
}
