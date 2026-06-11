package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestSwarmSectionOptions verifies swarm.Options construction from config values.
*/
func TestSwarmSectionOptions(t *testing.T) {
	Convey("Given swarm defaults", t, func() {
		section := SwarmSection{
			MeshID:           "swarm",
			GossipTTLSeconds: 30,
			MeshTTLSeconds:   5400,
			MeshBuffer:       64,
		}

		Convey("When Options is called", func() {
			options, err := section.Options()

			Convey("Then it should return mesh settings from config", func() {
				So(err, ShouldBeNil)
				So(options.MeshID, ShouldEqual, "swarm")
				So(options.GossipTTL, ShouldEqual, 30*time.Second)
				So(options.MeshTTL, ShouldEqual, 5400*time.Second)
				So(options.Buffer, ShouldEqual, 64)
			})
		})
	})

	Convey("Given missing mesh_id", t, func() {
		section := SwarmSection{
			GossipTTLSeconds: 30,
			MeshTTLSeconds:   5400,
			MeshBuffer:       64,
		}

		Convey("When Options is called", func() {
			_, err := section.Options()

			Convey("Then it should reject the missing mesh ID", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}
