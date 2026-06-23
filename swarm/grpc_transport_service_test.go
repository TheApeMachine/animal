package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

/*
TestMeshJSONCodec verifies the gRPC transport codec preserves envelopes.
*/
func TestMeshJSONCodec(t *testing.T) {
	Convey("Given a mesh envelope and JSON codec", t, func() {
		rumor := NewRumorAt(
			KindStatus,
			"actor-a",
			"Ada",
			"developer",
			time.Unix(100, 0),
		)
		rumor.State = "idle"

		envelope, err := NewMeshEnvelope(
			"grpc-codec-test",
			"actor-a",
			MessageTypeRumor,
			rumor,
		)
		So(err, ShouldBeNil)

		codec := meshJSONCodec{}

		Convey("When the envelope is marshaled and unmarshaled", func() {
			data, err := codec.Marshal(envelope)
			So(err, ShouldBeNil)

			decoded := MeshEnvelope{}
			err = codec.Unmarshal(data, &decoded)

			Convey("Then the transport envelope should survive the round trip", func() {
				So(err, ShouldBeNil)
				So(codec.Name(), ShouldEqual, "json")
				So(decoded.MeshID, ShouldEqual, envelope.MeshID)
				So(decoded.SenderID, ShouldEqual, envelope.SenderID)
				So(decoded.MessageType, ShouldEqual, envelope.MessageType)
				So(string(decoded.Payload), ShouldEqual, string(envelope.Payload))
			})
		})
	})
}
