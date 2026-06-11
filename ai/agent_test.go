package ai

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/spf13/viper"
	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/animal/swarm"
	"github.com/theapemachine/qpool"
)

func configureAgentTestViper() {
	viper.Set("ai.prompt.template.system", "You are {{ agent.name }}, a {{ agent.role }}.")
	viper.Set("project.name", "Animal")
	viper.Set("project.description", "Multi-agent coordination harness.")
}

/*
TestAgentHandleIncoming verifies swarm rumors and provider messages in Cycle.
*/
func TestAgentHandleIncoming(t *testing.T) {
	Convey("Given a swarm-attached agent", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		registry, err := swarm.NewRegistry(ctx, pool, swarm.Options{
			MeshID:    "agent-test-mesh",
			GossipTTL: 30 * time.Second,
			MeshTTL:   time.Minute,
			Buffer:    8,
		}, lease.Options{
			KeySpace: lease.PathKeySpace{},
			IdleTTL:  time.Minute,
		})
		So(err, ShouldBeNil)

		agent, err := NewAgent(ctx, pool, "developer", "Ada", registry, []string{"lanes/a/"})
		So(err, ShouldBeNil)

		peer, err := registry.NewParticipant("peer-b", "Bob", "developer", nil)
		So(err, ShouldBeNil)

		announce := swarm.NewRumorAt(swarm.KindAnnounce, "peer-b", "Bob", "developer", time.Now())
		announce.Topic = "roadmap.announce"
		announce.Payload = "ship leases first"

		Convey("When a peer publishes an announce rumor", func() {
			publishErr := peer.Announce(announce.Topic, announce.Payload)
			So(publishErr, ShouldBeNil)

			var qv *qpool.QValue[any]
			deadline := time.Now().Add(time.Second)

			for time.Now().Before(deadline) {
				qv = agent.pollIncoming()
				if qv != nil {
					break
				}

				time.Sleep(time.Millisecond)
			}

			if qv == nil {
				t.Fatal("timed out waiting for announce rumor")
			}

			agent.handleIncoming(qv)

			Convey("Then the agent view should contain the announcement", func() {
				records := agent.Participant().View().RecentAnnounces()
				So(len(records), ShouldEqual, 1)
				So(records[0].Topic, ShouldEqual, "roadmap.announce")
			})
		})
	})

	Convey("Given a legacy agent without swarm", t, func() {
		configureAgentTestViper()
		ctx := context.Background()
		pool := qpool.NewQ[any](ctx, 1, 1, &qpool.Config{Scaler: nil})

		agent, err := NewAgent(ctx, pool, "developer", "Ada", nil, nil)
		So(err, ShouldBeNil)

		message := provider.Message{ID: "m-1", Role: "user", Content: "hello"}

		Convey("When a provider message arrives", func() {
			agent.handleIncoming(&qpool.QValue[any]{Value: message})

			Convey("Then it should append to agent context messages", func() {
				So(len(agent.Context.Messages), ShouldEqual, 1)
				So(agent.Context.Messages[0].Content, ShouldEqual, "hello")
			})
		})
	})
}
