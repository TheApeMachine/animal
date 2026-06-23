package swarm

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/theapemachine/datura"
)

/*
TestNewMeshEnvelope verifies typed swarm values can enter the transport envelope.
*/
func TestNewMeshEnvelope(t *testing.T) {
	Convey("Given a typed rumor", t, func() {
		rumor := NewRumorAt(
			KindStatus,
			"actor-a",
			"Ada",
			"developer",
			time.Unix(100, 0),
		)
		rumor.State = "idle"

		Convey("When an envelope is created", func() {
			envelope, err := NewMeshEnvelope(
				"transport-test",
				"actor-a",
				MessageTypeRumor,
				rumor,
			)

			Convey("Then the transport fields and payload should be present", func() {
				So(err, ShouldBeNil)
				So(envelope.MeshID, ShouldEqual, "transport-test")
				So(envelope.SenderID, ShouldEqual, "actor-a")
				So(envelope.MessageType, ShouldEqual, MessageTypeRumor)
				So(len(envelope.Payload), ShouldBeGreaterThan, 0)
			})
		})
	})
}

/*
TestMeshEnvelopeArtifact verifies remote envelopes re-enter the qpool artifact path.
*/
func TestMeshEnvelopeArtifact(t *testing.T) {
	Convey("Given a transport envelope with a rumor payload", t, func() {
		rumor := NewRumorAt(
			KindStatus,
			"actor-a",
			"Ada",
			"developer",
			time.Unix(100, 0),
		)
		rumor.State = "working"

		envelope, err := NewMeshEnvelope(
			"transport-test",
			"actor-a",
			MessageTypeRumor,
			rumor,
		)
		So(err, ShouldBeNil)

		Convey("When the envelope is converted to a qpool artifact", func() {
			artifact, err := envelope.Artifact(time.Second)
			So(err, ShouldBeNil)

			payload := datura.As[Rumor](artifact)

			Convey("Then participants should decode the original typed rumor", func() {
				So(payload.ActorID, ShouldEqual, "actor-a")
				So(payload.State, ShouldEqual, "working")
			})
		})
	})
}

/*
TestMeshEnvelopeValue verifies unsupported message types are rejected.
*/
func TestMeshEnvelopeValue(t *testing.T) {
	Convey("Given an envelope with an unsupported message type", t, func() {
		envelope := MeshEnvelope{
			MeshID:      "transport-test",
			SenderID:    "actor-a",
			MessageType: "unknown",
			Payload:     []byte(`{"ok":true}`),
		}

		Convey("When the payload is decoded", func() {
			value, err := envelope.Value()

			Convey("Then decoding should fail without a fallback type", func() {
				So(value, ShouldBeNil)
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func BenchmarkMeshEnvelopeArtifact(benchmark *testing.B) {
	rumor := NewRumorAt(
		KindStatus,
		"actor-a",
		"Ada",
		"developer",
		time.Unix(100, 0),
	)
	rumor.State = "working"

	envelope, err := NewMeshEnvelope(
		"transport-test",
		"actor-a",
		MessageTypeRumor,
		rumor,
	)

	if err != nil {
		benchmark.Fatal(err)
	}

	benchmark.ReportAllocs()

	for benchmark.Loop() {
		if _, err := envelope.Artifact(time.Second); err != nil {
			benchmark.Fatal(err)
		}
	}
}
