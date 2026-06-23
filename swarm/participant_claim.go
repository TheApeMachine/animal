package swarm

import (
	"fmt"
	"time"

	"github.com/theapemachine/errnie"
)

/*
TryClaimConfigured attempts configured prefixes with the default contention backoff.
*/
func (participant *Participant) TryClaimConfigured() (string, error) {
	backoff, err := DefaultContentionBackoff(participant.ctx)

	if err != nil {
		return "", err
	}

	return participant.TryClaimConfiguredWithBackoff(backoff)
}

/*
TryClaimConfiguredWithBackoff retries coordinator-rejected claims with jittered backoff.
*/
func (participant *Participant) TryClaimConfiguredWithBackoff(
	backoff *ContentionBackoff,
) (string, error) {
	if backoff == nil {
		return "", errnie.Err(errnie.Validation, "swarm contention backoff is required", nil)
	}

	var err error

	for attempt := 1; attempt <= backoff.Attempts(); attempt++ {
		var prefix string
		var claimed bool
		prefix, claimed, err = participant.tryConfiguredOnce()

		if err == nil && claimed {
			return prefix, nil
		}

		if err == nil {
			break
		}

		if attempt == backoff.Attempts() {
			break
		}

		if err = backoff.Wait(attempt); err != nil {
			return "", err
		}
	}

	return "", errnie.Error(errnie.Err(
		errnie.NotFound,
		fmt.Sprintf(
			"no configured prefix available for actor %q",
			participant.actorID,
		),
		err,
	))
}

func (participant *Participant) tryConfiguredOnce() (string, bool, error) {
	participant.view.PurgeExpired(time.Now())

	var err error

	for _, prefix := range participant.claimPrefixes {
		if !participant.view.IsPrefixFree(prefix) {
			continue
		}

		if err = participant.TryClaim(prefix); err != nil {
			continue
		}

		return prefix, true, nil
	}

	return "", false, err
}
