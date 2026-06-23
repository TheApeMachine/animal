package swarm

import (
	"fmt"
	"strings"
	"time"

	"github.com/theapemachine/errnie"
)

/*
TaskClaim is an optimistic claim on an A2A task before work starts.
*/
type TaskClaim struct {
	TaskID       string
	ActorID      string
	ActorName    string
	Role         string
	At           int64
	ConfirmAfter int64
}

/*
NewTaskClaimAt builds a claim with an explicit confirmation window.
*/
func NewTaskClaimAt(
	taskID string,
	actorID string,
	actorName string,
	role string,
	at time.Time,
	confirmationWindow time.Duration,
) (TaskClaim, error) {
	if confirmationWindow <= 0 {
		return TaskClaim{}, errnie.Err(
			errnie.Validation,
			"swarm task claim confirmation window is required",
			nil,
		)
	}

	claim := TaskClaim{
		TaskID:       taskID,
		ActorID:      actorID,
		ActorName:    actorName,
		Role:         role,
		At:           at.UnixNano(),
		ConfirmAfter: at.Add(confirmationWindow).UnixNano(),
	}

	return claim, claim.Validate()
}

/*
Validate checks required task claim fields.
*/
func (claim TaskClaim) Validate() error {
	if strings.TrimSpace(claim.TaskID) == "" {
		return errnie.Err(errnie.Validation, "swarm task claim task id is required", nil)
	}

	if strings.TrimSpace(claim.ActorID) == "" {
		return errnie.Err(errnie.Validation, "swarm task claim actor id is required", nil)
	}

	if claim.At <= 0 {
		return errnie.Err(errnie.Validation, "swarm task claim timestamp is required", nil)
	}

	if claim.ConfirmAfter <= claim.At {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm task claim confirm_after must be after %d", claim.At),
			nil,
		)
	}

	return nil
}

/*
Ready reports whether the confirmation window has elapsed.
*/
func (claim TaskClaim) Ready(now time.Time) bool {
	return now.UnixNano() >= claim.ConfirmAfter
}
