package swarm

import (
	"context"
	"fmt"

	"github.com/theapemachine/errnie"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
Publish sends an envelope to every configured gRPC peer.
*/
func (transport *GRPCMeshTransport) Publish(
	ctx context.Context,
	envelope MeshEnvelope,
) error {
	if err := envelope.Validate(); err != nil {
		return err
	}

	for _, peer := range transport.peers {
		if err := transport.publishPeer(ctx, peer, envelope); err != nil {
			return err
		}
	}

	return nil
}

func (transport *GRPCMeshTransport) publishPeer(
	ctx context.Context,
	peer string,
	envelope MeshEnvelope,
) error {
	client, err := grpc.NewClient(
		peer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(meshJSONCodec{})),
	)

	if err != nil {
		return errnie.Err(errnie.Network, "swarm grpc mesh client failed", err)
	}

	reply := &MeshAck{}
	err = client.Invoke(ctx, grpcMeshPublishPath, &envelope, reply)

	if err != nil {
		return errnie.Combine(
			errnie.Err(
				errnie.Network,
				fmt.Sprintf("swarm grpc mesh publish to %q failed", peer),
				err,
			),
			errnie.Guard(
				errnie.Network,
				fmt.Sprintf("swarm grpc mesh close peer %q failed", peer),
				client.Close(),
			),
		)
	}

	if err := client.Close(); err != nil {
		return errnie.Err(
			errnie.Network,
			fmt.Sprintf("swarm grpc mesh close peer %q failed", peer),
			err,
		)
	}

	if !reply.Accepted {
		return errnie.Err(
			errnie.Validation,
			fmt.Sprintf("swarm grpc mesh peer %q rejected envelope", peer),
			nil,
		)
	}

	return nil
}
