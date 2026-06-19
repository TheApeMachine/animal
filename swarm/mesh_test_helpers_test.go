package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/theapemachine/datura"
	"github.com/theapemachine/qpool"
)

func waitBroadcastConsumer(
	ctx context.Context,
	consumer *qpool.BroadcastConsumer,
	timeout time.Duration,
) (*datura.Artifact, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if artifact := consumer.Poll(); artifact != nil {
			return artifact, nil
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
) (*datura.Artifact, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if artifact := participant.Poll(); artifact != nil {
			return artifact, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}

	return nil, fmt.Errorf("swarm: timed out waiting for participant message")
}
