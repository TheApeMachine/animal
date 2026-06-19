package internal

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/theapemachine/qpool"
)

func newTestBus(test *testing.T) *Bus {
	test.Helper()

	ctx := context.Background()
	pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

	viper.Set("system.queue.ttl", time.Minute)

	return NewBus(
		ctx,
		pool,
		[]Channel{ChannelMessages},
		[]Subscription{Subscribe(ChannelMessages, "test-agent")},
	)
}

func TestNewBus(test *testing.T) {
	Convey("Given ChannelMessages broadcast registration", test, func() {
		bus := newTestBus(test)

		Convey("When NewBus initializes the route", func() {
			Convey("Then the messages broadcast group should exist", func() {
				So(bus.broadcasts[ChannelMessages], ShouldNotBeNil)
				So(bus.subscribers[ChannelMessages], ShouldNotBeNil)
			})
		})
	})
}

func TestBusSendReceive(test *testing.T) {
	Convey("Given a subscribed bus", test, func() {
		bus := newTestBus(test)

		Convey("When Send publishes a message", func() {
			err := bus.Send(ChannelMessages, "ping", "hello")

			So(err, ShouldBeNil)

			Convey("Then Receive should deliver the artifact", func() {
				artifact, receiveErr := bus.Receive(ChannelMessages)

				So(receiveErr, ShouldBeNil)
				So(artifact, ShouldNotBeNil)
			})
		})
	})
}

func TestBusPoll(test *testing.T) {
	Convey("Given a subscribed bus with a queued message", test, func() {
		bus := newTestBus(test)

		err := bus.Send(ChannelMessages, "poll", map[string]string{"key": "value"})

		So(err, ShouldBeNil)

		Convey("When Poll is called on the subscribed channel", func() {
			artifact, pollErr := bus.Poll(ChannelMessages)

			So(pollErr, ShouldBeNil)
			So(artifact, ShouldNotBeNil)
		})
	})
}

func TestBusReceiveUnknownChannel(test *testing.T) {
	Convey("Given a bus without the requested subscription", test, func() {
		bus := newTestBus(test)

		Convey("When Receive is called on an unknown channel", func() {
			artifact, err := bus.Receive(Channel("unknown"))

			So(artifact, ShouldBeNil)
			So(err, ShouldNotBeNil)
		})
	})
}
