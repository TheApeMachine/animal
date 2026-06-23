package swarm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/theapemachine/errnie"
)

/*
LeaseHeartbeat periodically renews one participant lease and republishes its claim.
*/
type LeaseHeartbeat struct {
	ctx         context.Context
	cancel      context.CancelFunc
	err         error
	participant *Participant
	prefix      string
	interval    time.Duration
	done        chan error
	started     atomic.Bool
}

/*
NewLeaseHeartbeat instantiates a periodic lease renewer for one participant prefix.
*/
func NewLeaseHeartbeat(
	ctx context.Context,
	participant *Participant,
	prefix string,
	interval time.Duration,
) (*LeaseHeartbeat, error) {
	if interval <= 0 {
		return nil, errnie.Err(errnie.Validation, "swarm heartbeat interval is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	heartbeat := &LeaseHeartbeat{
		ctx:         ctx,
		cancel:      cancel,
		participant: participant,
		prefix:      prefix,
		interval:    interval,
		done:        make(chan error, 1),
	}

	return heartbeat, errnie.Require(map[string]any{
		"ctx":         heartbeat.ctx,
		"cancel":      heartbeat.cancel,
		"participant": heartbeat.participant,
		"prefix":      heartbeat.prefix,
		"interval":    heartbeat.interval,
		"done":        heartbeat.done,
	})
}

/*
Start renews the lease immediately and then on every interval tick.
*/
func (heartbeat *LeaseHeartbeat) Start() error {
	if !heartbeat.started.CompareAndSwap(false, true) {
		return errnie.Err(errnie.Validation, "swarm heartbeat already started", nil)
	}

	if err := heartbeat.participant.Renew(heartbeat.prefix); err != nil {
		heartbeat.started.Store(false)

		return err
	}

	go heartbeat.loop()

	return nil
}

/*
Stop cancels the heartbeat and waits for the loop to exit.
*/
func (heartbeat *LeaseHeartbeat) Stop() error {
	heartbeat.cancel()

	return heartbeat.Wait()
}

/*
Wait blocks until the heartbeat loop exits.
*/
func (heartbeat *LeaseHeartbeat) Wait() error {
	if !heartbeat.started.Load() {
		return errnie.Err(errnie.Validation, "swarm heartbeat is not started", nil)
	}

	return <-heartbeat.done
}

func (heartbeat *LeaseHeartbeat) loop() {
	ticker := time.NewTicker(heartbeat.interval)
	defer ticker.Stop()

	for {
		select {
		case <-heartbeat.ctx.Done():
			heartbeat.done <- nil
			close(heartbeat.done)

			return
		case <-ticker.C:
			if err := heartbeat.participant.Renew(heartbeat.prefix); err != nil {
				heartbeat.done <- err
				close(heartbeat.done)

				return
			}
		}
	}
}

/*
LeaseSweeper periodically publishes release rumors for expired coordinator leases.
*/
type LeaseSweeper struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	registry *Registry
	interval time.Duration
	done     chan error
	started  atomic.Bool
}

/*
NewLeaseSweeper instantiates a registry lease-expiration broadcaster.
*/
func NewLeaseSweeper(
	ctx context.Context,
	registry *Registry,
	interval time.Duration,
) (*LeaseSweeper, error) {
	if interval <= 0 {
		return nil, errnie.Err(errnie.Validation, "swarm sweeper interval is required", nil)
	}

	ctx, cancel := context.WithCancel(ctx)

	sweeper := &LeaseSweeper{
		ctx:      ctx,
		cancel:   cancel,
		registry: registry,
		interval: interval,
		done:     make(chan error, 1),
	}

	return sweeper, errnie.Require(map[string]any{
		"ctx":      sweeper.ctx,
		"cancel":   sweeper.cancel,
		"registry": sweeper.registry,
		"interval": sweeper.interval,
		"done":     sweeper.done,
	})
}

/*
Start sweeps once immediately and then on every interval tick.
*/
func (sweeper *LeaseSweeper) Start() error {
	if !sweeper.started.CompareAndSwap(false, true) {
		return errnie.Err(errnie.Validation, "swarm sweeper already started", nil)
	}

	if _, err := sweeper.registry.SweepExpiredLeases(); err != nil {
		sweeper.started.Store(false)

		return err
	}

	go sweeper.loop()

	return nil
}

/*
Stop cancels the sweeper and waits for the loop to exit.
*/
func (sweeper *LeaseSweeper) Stop() error {
	sweeper.cancel()

	return sweeper.Wait()
}

/*
Wait blocks until the sweeper loop exits.
*/
func (sweeper *LeaseSweeper) Wait() error {
	if !sweeper.started.Load() {
		return errnie.Err(errnie.Validation, "swarm sweeper is not started", nil)
	}

	return <-sweeper.done
}

func (sweeper *LeaseSweeper) loop() {
	ticker := time.NewTicker(sweeper.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sweeper.ctx.Done():
			sweeper.done <- nil
			close(sweeper.done)

			return
		case <-ticker.C:
			if _, err := sweeper.registry.SweepExpiredLeases(); err != nil {
				sweeper.done <- err
				close(sweeper.done)

				return
			}
		}
	}
}

/*
SweepExpiredLeases removes stale leases and broadcasts release rumors for peers.
*/
func (registry *Registry) SweepExpiredLeases() ([]string, error) {
	sweepAt := time.Now()
	expired, err := registry.coordinator.SweepExpired(sweepAt)

	if err != nil {
		return nil, err
	}

	prefixes := make([]string, 0, len(expired))

	for _, record := range expired {
		rumor := NewRumorAt(KindRelease, record.ActorID, "", "", sweepAt)
		rumor.Prefix = record.Prefix

		if err := registry.mesh.Publish(record.ActorID, rumor); err != nil {
			return prefixes, fmt.Errorf("swarm: publish expired lease release: %w", err)
		}

		prefixes = append(prefixes, record.Prefix)
	}

	return prefixes, nil
}
