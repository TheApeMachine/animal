package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/theapemachine/qpool"
)

func waitBroadcastConsumer(
	ctx context.Context,
	consumer *qpool.BroadcastConsumer,
	timeout time.Duration,
) (*qpool.QValue[any], error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if qv := consumer.Poll(); qv != nil {
			return qv, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}

	return nil, fmt.Errorf("swarm: timed out waiting for broadcast message")
}

func waitParticipant(
	ctx context.Context,
	participant *Participant,
	timeout time.Duration,
) (*qpool.QValue[any], error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if qv := participant.Poll(); qv != nil {
			return qv, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}

	return nil, fmt.Errorf("swarm: timed out waiting for participant message")
}
