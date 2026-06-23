package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/theapemachine/animal/a2a"
	"github.com/theapemachine/datura"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
MeshReceiveFunc accepts one remote mesh envelope.
*/
type MeshReceiveFunc func(MeshEnvelope) error

/*
MeshTransport publishes mesh envelopes outside the local qpool process.
*/
type MeshTransport interface {
	Start(ctx context.Context, receive MeshReceiveFunc) error
	Publish(ctx context.Context, envelope MeshEnvelope) error
	Close() error
}

/*
MeshEnvelope carries a typed swarm artifact over an external transport.
*/
type MeshEnvelope struct {
	MeshID      string
	SenderID    string
	MessageType string
	Payload     json.RawMessage
}

/*
NewMeshEnvelope serializes a typed swarm value for local or remote delivery.
*/
func NewMeshEnvelope(
	meshID string,
	senderID string,
	messageType string,
	value any,
) (MeshEnvelope, error) {
	if value == nil {
		return MeshEnvelope{}, errnie.Err(
			errnie.Validation,
			"swarm mesh envelope value is required",
			nil,
		)
	}

	payload, err := json.Marshal(value)

	if err != nil {
		return MeshEnvelope{}, err
	}

	envelope := MeshEnvelope{
		MeshID:      meshID,
		SenderID:    senderID,
		MessageType: messageType,
		Payload:     payload,
	}

	return envelope, envelope.Validate()
}

/*
Validate checks envelope routing fields before transport delivery.
*/
func (envelope MeshEnvelope) Validate() error {
	if envelope.MeshID == "" {
		return errnie.Err(errnie.Validation, "swarm mesh id is required", nil)
	}

	if envelope.SenderID == "" {
		return errnie.Err(errnie.Validation, "swarm mesh sender id is required", nil)
	}

	if envelope.MessageType == "" {
		return errnie.Err(errnie.Validation, "swarm mesh message type is required", nil)
	}

	if len(envelope.Payload) == 0 {
		return errnie.Err(errnie.Validation, "swarm mesh payload is required", nil)
	}

	return nil
}

/*
Artifact converts the envelope into the qpool artifact participants already consume.
*/
func (envelope MeshEnvelope) Artifact(ttl time.Duration) (*datura.Artifact, error) {
	if ttl <= 0 {
		return nil, errnie.Err(errnie.Validation, "swarm mesh ttl is required", nil)
	}

	value, err := envelope.Value()

	if err != nil {
		return nil, err
	}

	return qpool.NewBusArtifact(
		envelope.SenderID,
		envelope.SenderID,
		envelope.MessageType,
		value,
		ttl,
	)
}

/*
Value decodes the envelope payload into its concrete swarm type.
*/
func (envelope MeshEnvelope) Value() (any, error) {
	if err := envelope.Validate(); err != nil {
		return nil, err
	}

	switch envelope.MessageType {
	case MessageTypeRumor:
		return meshEnvelopeValue[Rumor](envelope.Payload)
	case MessageTypeTask:
		return meshEnvelopeValue[a2a.Task](envelope.Payload)
	case MessageTypeTaskClaim:
		return meshEnvelopeValue[TaskClaim](envelope.Payload)
	case MessageTypeTaskStatus:
		return meshEnvelopeValue[a2a.TaskStatusUpdateEvent](envelope.Payload)
	case MessageTypeSignal:
		return meshEnvelopeValue[Signal](envelope.Payload)
	case MessageTypeMetric:
		return meshEnvelopeValue[Metric](envelope.Payload)
	case MessageTypeContention:
		return meshEnvelopeValue[Contention](envelope.Payload)
	default:
		return nil, errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm mesh message type %q is unsupported", envelope.MessageType),
			nil,
		)
	}
}

func meshEnvelopeValue[Value any](payload json.RawMessage) (Value, error) {
	var value Value

	err := json.Unmarshal(payload, &value)

	if err != nil {
		return value, err
	}

	return value, nil
}
