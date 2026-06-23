package swarm

import (
	"context"
	"time"

	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
ContentionBackoff calculates jittered delays for rejected lease claim attempts.
*/
type ContentionBackoff struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	attempts int
	initial  time.Duration
	max      time.Duration
	jitter   time.Duration
	sleep    func(context.Context, time.Duration) error
	random   func() uint32
}

/*
NewContentionBackoff instantiates a retry delay policy for lease contention.
*/
func NewContentionBackoff(
	ctx context.Context,
	attempts int,
	initial time.Duration,
	max time.Duration,
	jitter time.Duration,
) (*ContentionBackoff, error) {
	if attempts <= 0 {
		return nil, errnie.Err(errnie.Validation, "swarm backoff attempts are required", nil)
	}

	if initial <= 0 {
		return nil, errnie.Err(errnie.Validation, "swarm backoff initial delay is required", nil)
	}

	if max < initial {
		return nil, errnie.Err(errnie.Validation, "swarm backoff max delay is too small", nil)
	}

	if jitter < 0 {
		return nil, errnie.Err(errnie.Validation, "swarm backoff jitter must be non-negative", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	backoff := &ContentionBackoff{
		ctx:      ctx,
		cancel:   cancel,
		attempts: attempts,
		initial:  initial,
		max:      max,
		jitter:   jitter,
		sleep:    sleepBackoff,
		random:   qpool.Fastrand,
	}

	return backoff, errnie.Require(map[string]any{
		"ctx":      backoff.ctx,
		"cancel":   backoff.cancel,
		"attempts": backoff.attempts,
		"initial":  backoff.initial,
		"max":      backoff.max,
		"sleep":    backoff.sleep,
		"random":   backoff.random,
	})
}

/*
DefaultContentionBackoff returns the standard claim retry policy.
*/
func DefaultContentionBackoff(ctx context.Context) (*ContentionBackoff, error) {
	return NewContentionBackoff(
		ctx,
		3,
		5*time.Millisecond,
		50*time.Millisecond,
		5*time.Millisecond,
	)
}

/*
Attempts returns the maximum number of claim attempts.
*/
func (backoff *ContentionBackoff) Attempts() int {
	return backoff.attempts
}

/*
Delay returns the jittered delay for one failed attempt.
*/
func (backoff *ContentionBackoff) Delay(attempt int) (time.Duration, error) {
	if attempt <= 0 {
		return 0, errnie.Err(errnie.Validation, "swarm backoff attempt is required", nil)
	}

	delay := (&qpool.ExponentialBackoff{Initial: backoff.initial}).NextDelay(attempt)

	if delay > backoff.max {
		delay = backoff.max
	}

	if backoff.jitter == 0 {
		return delay, nil
	}

	span := uint64(backoff.jitter / time.Nanosecond)
	if span == 0 {
		return delay, nil
	}

	return delay + time.Duration(uint64(backoff.random())%(span+1))*time.Nanosecond, nil
}

/*
Wait sleeps for the calculated delay unless the backoff context is canceled.
*/
func (backoff *ContentionBackoff) Wait(attempt int) error {
	delay, err := backoff.Delay(attempt)

	if err != nil {
		return err
	}

	return backoff.sleep(backoff.ctx, delay)
}

func sleepBackoff(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
