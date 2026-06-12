package internal

import (
	"context"
	"fmt"

	"github.com/spf13/viper"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Bus routes typed qpool broadcast messages between agents on
named channels. It centralizes channel registration so producers
and consumers share one pool-backed fan-out surface.
*/
type Bus struct {
	ctx         context.Context
	cancel      context.CancelFunc
	err         error
	pool        *qpool.Q[any]
	broadcasts  map[Channel]*qpool.BroadcastGroup
	subscribers map[Channel]*qpool.BroadcastConsumer
}

/*
NewBus creates a new bus with a context, pool, broadcasts, and subscriptions.
*/
func NewBus(
	ctx context.Context,
	pool *qpool.Q[any],
	broadcasts []Channel,
	subscriptions []Subscription,
) *Bus {
	ctx, cancel := context.WithCancel(ctx)

	bus := &Bus{
		ctx:         ctx,
		cancel:      cancel,
		pool:        pool,
		broadcasts:  make(map[Channel]*qpool.BroadcastGroup),
		subscribers: make(map[Channel]*qpool.BroadcastConsumer),
	}

	for _, broadcast := range broadcasts {
		bus.broadcasts[broadcast] = pool.CreateBroadcastGroup(
			broadcast.String(),
			viper.GetDuration("system.queue.ttl"),
		)
	}

	for _, subscription := range subscriptions {
		if bus.broadcasts[subscription.Channel] == nil {
			bus.broadcasts[subscription.Channel] = pool.CreateBroadcastGroup(
				subscription.Channel.String(),
				viper.GetDuration("system.queue.ttl"),
			)
		}

		subscriberName := subscription.Name

		if subscriberName == "" {
			subscriberName = subscription.Channel.String()
		}

		bus.subscribers[subscription.Channel] = bus.broadcasts[subscription.Channel].Subscribe(
			subscriberName,
			viper.GetInt("system.queue.buffer"),
		)
	}

	return bus
}

/*
Receive blocks until the next message is available on the given channel.
*/
func (bus *Bus) Receive(channel Channel) (*qpool.QValue[any], error) {
	if bus.subscribers[channel] == nil {
		return nil, errnie.Err(
			errnie.Validation,
			fmt.Sprintf("bus receive channel %s not found", channel),
			nil,
		)
	}

	return bus.subscribers[channel].Wait(bus.ctx)
}

/*
Poll returns the next message on the given channel without blocking.
*/
func (bus *Bus) Poll(channel Channel) (*qpool.QValue[any], error) {
	if bus.subscribers[channel] == nil {
		return nil, errnie.Err(
			errnie.Validation,
			fmt.Sprintf("bus receive channel %s not found", channel),
			nil,
		)
	}

	return bus.subscribers[channel].Poll(), nil
}

/*
Send publishes a message on the given channel.
*/
func (bus *Bus) Send(
	channel Channel, messageType string, value any,
) error {
	if bus.broadcasts[channel] == nil {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("bus send channel %s not found", channel),
			nil,
		)
	}

	bus.broadcasts[channel].Send(&qpool.QValue[any]{
		Type:  messageType,
		Value: value,
	})

	return nil
}

/*
Close stops the bus and cancels its context.
*/
func (bus *Bus) Close() error {
	bus.cancel()
	return nil
}
