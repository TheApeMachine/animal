package conversation

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/animal/swarm"
)

/*
TestSalonRegistryApply verifies transcript and stance merge from gossip.
*/
func TestSalonRegistryApply(t *testing.T) {
	Convey("Given a fresh salon registry", t, func() {
		registry := NewSalonRegistry()

		turn := swarm.AnnounceRecord{
			ActorID:   "actor-a",
			ActorName: "Ada",
			Role:      "developer",
			Topic:     TopicTurn,
			Payload:   `{"content":"hello salon"}`,
			At:        100,
		}

		stance := swarm.AnnounceRecord{
			ActorID: "actor-a",
			Topic:   TopicStance,
			Payload: `{"themes":["governance","oversight"]}`,
			At:      101,
		}

		Convey("When turn and stance announces are applied", func() {
			So(registry.Apply(turn), ShouldBeNil)
			So(registry.Apply(stance), ShouldBeNil)

			Convey("Then the transcript and themes should be stored", func() {
				So(len(registry.Turns()), ShouldEqual, 1)
				So(registry.ThemeSet("actor-a"), ShouldContainKey, "governance")
				So(registry.ThemeSet("actor-a"), ShouldContainKey, "oversight")
			})
		})
	})
}

/*
TestSplitReplyAndStance verifies STANCE line parsing from model output.
*/
func TestSplitReplyAndStance(t *testing.T) {
	Convey("Given a reply with a trailing stance line", t, func() {
		reply := "We need dialogue first.\nSTANCE: governance, dialogue, caution"

		Convey("When SplitReplyAndStance is called", func() {
			spoken, themes := SplitReplyAndStance(reply)

			Convey("Then it should split speech from themes", func() {
				So(spoken, ShouldEqual, "We need dialogue first.")
				So(themes, ShouldResemble, []string{"governance", "dialogue", "caution"})
			})
		})
	})
}

/*
TestComputeClusters verifies emergent alignment grouping from theme overlap.
*/
func TestComputeClusters(t *testing.T) {
	Convey("Given three actors with overlapping stances", t, func() {
		registry := NewSalonRegistry()
		names := map[string]string{
			"a": "Ada",
			"b": "Bob",
			"c": "Quinn",
		}

		So(registry.Apply(swarm.AnnounceRecord{
			ActorID: "a", Topic: TopicStance,
			Payload: `{"themes":["governance","oversight"]}`, At: 1,
		}), ShouldBeNil)
		So(registry.Apply(swarm.AnnounceRecord{
			ActorID: "b", Topic: TopicStance,
			Payload: `{"themes":["governance","oversight"]}`, At: 2,
		}), ShouldBeNil)
		So(registry.Apply(swarm.AnnounceRecord{
			ActorID: "c", Topic: TopicStance,
			Payload: `{"themes":["testing","risk"]}`, At: 3,
		}), ShouldBeNil)

		So(registry.Apply(swarm.AnnounceRecord{
			ActorID: "a", ActorName: "Ada", Topic: TopicTurn,
			Payload: `{"content":"hi"}`, At: 4,
		}), ShouldBeNil)

		Convey("When clusters are computed", func() {
			clusters := ComputeClusters(registry, names, 0.34)

			Convey("Then governance-aligned speakers should cluster on distinctive overlap", func() {
				So(len(clusters), ShouldEqual, 1)
				So(clusters[0].Members, ShouldContain, "Ada")
				So(clusters[0].Members, ShouldContain, "Bob")
				So(clusters[0].Members, ShouldNotContain, "Quinn")
			})
		})
	})

	Convey("Given universal themes across the whole panel", t, func() {
		registry := NewSalonRegistry()
		names := map[string]string{
			"a": "Ada",
			"b": "Bob",
			"c": "Quinn",
			"d": "Dana",
		}

		for index, actorID := range []string{"a", "b", "c", "d"} {
			So(registry.Apply(swarm.AnnounceRecord{
				ActorID: actorID, Topic: TopicStance,
				Payload: `{"themes":["governance","dialogue"]}`, At: int64(index + 10),
			}), ShouldBeNil)
		}

		Convey("When clusters are computed", func() {
			clusters := ComputeClusters(registry, names, 0.34)

			Convey("Then universal themes should not produce a megacluster", func() {
				So(len(clusters), ShouldEqual, 0)
			})
		})
	})
}
