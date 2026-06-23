package swarm

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
)

/*
MeshAck confirms one remote publish request.
*/
type MeshAck struct {
	Accepted bool
}

type grpcMeshReceiver interface {
	Receive(context.Context, *MeshEnvelope) (*MeshAck, error)
}

var grpcMeshTransportService = grpc.ServiceDesc{
	ServiceName: "animal.swarm.MeshTransport",
	HandlerType: (*grpcMeshReceiver)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Publish",
			Handler:    grpcMeshPublishHandler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "animal/swarm/grpc_transport",
}

func grpcMeshPublishHandler(
	server any,
	ctx context.Context,
	decode func(any) error,
	interceptor grpc.UnaryServerInterceptor,
) (any, error) {
	envelope := new(MeshEnvelope)

	if err := decode(envelope); err != nil {
		return nil, err
	}

	if interceptor == nil {
		return server.(grpcMeshReceiver).Receive(ctx, envelope)
	}

	info := &grpc.UnaryServerInfo{
		Server:     server,
		FullMethod: grpcMeshPublishPath,
	}
	handler := func(ctx context.Context, request any) (any, error) {
		return server.(grpcMeshReceiver).Receive(ctx, request.(*MeshEnvelope))
	}

	return interceptor(ctx, envelope, info, handler)
}

type meshJSONCodec struct{}

func (meshJSONCodec) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (meshJSONCodec) Unmarshal(data []byte, value any) error {
	return json.Unmarshal(data, value)
}

func (meshJSONCodec) Name() string {
	return "json"
}
