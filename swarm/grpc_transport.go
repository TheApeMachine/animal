package swarm

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/theapemachine/errnie"
	"google.golang.org/grpc"
)

const grpcMeshPublishPath = "/animal.swarm.MeshTransport/Publish"

/*
GRPCMeshTransport publishes mesh envelopes to peer hosts over gRPC.
*/
type GRPCMeshTransport struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	listener net.Listener
	server   *grpc.Server
	peers    []string
	receive  MeshReceiveFunc
	stop     sync.Once
}

/*
NewGRPCMeshTransport opens a gRPC listener and stores outbound peers.
*/
func NewGRPCMeshTransport(
	ctx context.Context,
	listenAddress string,
	peers []string,
) (*GRPCMeshTransport, error) {
	if strings.TrimSpace(listenAddress) == "" {
		return nil, errnie.Err(
			errnie.Validation,
			"swarm grpc mesh listen address is required",
			nil,
		)
	}

	ctx, cancel := context.WithCancel(ctx)
	peerAddresses, err := cleanGRPCPeers(peers)

	if err != nil {
		cancel()

		return nil, err
	}

	listener, err := net.Listen("tcp", listenAddress)

	if err != nil {
		cancel()

		return nil, errnie.Err(errnie.Network, "swarm grpc mesh listen failed", err)
	}

	transport := &GRPCMeshTransport{
		ctx:      ctx,
		cancel:   cancel,
		listener: listener,
		peers:    peerAddresses,
	}

	err = errnie.Require(map[string]any{
		"ctx":      transport.ctx,
		"cancel":   transport.cancel,
		"listener": transport.listener,
	})

	if err != nil {
		cancel()

		return nil, errnie.Combine(
			err,
			errnie.Guard(
				errnie.Network,
				"swarm grpc mesh close listener failed",
				listener.Close(),
			),
		)
	}

	return transport, nil
}

/*
Start begins accepting remote mesh envelopes.
*/
func (transport *GRPCMeshTransport) Start(
	ctx context.Context,
	receive MeshReceiveFunc,
) error {
	if ctx == nil {
		return errnie.Err(errnie.Validation, "swarm grpc mesh context is required", nil)
	}

	if receive == nil {
		return errnie.Err(errnie.Validation, "swarm grpc mesh receiver is required", nil)
	}

	if transport.server != nil {
		return errnie.Err(errnie.Conflict, "swarm grpc mesh is already started", nil)
	}

	transport.receive = receive
	transport.server = grpc.NewServer(grpc.ForceServerCodec(meshJSONCodec{}))
	transport.server.RegisterService(&grpcMeshTransportService, transport)

	go transport.serve()
	go transport.stopWhenDone(ctx)

	return nil
}

/*
Close stops the listener and server.
*/
func (transport *GRPCMeshTransport) Close() error {
	transport.cancel()
	var err error

	transport.stop.Do(func() {
		if transport.server != nil {
			transport.server.GracefulStop()

			return
		}

		err = transport.listener.Close()
	})

	if err != nil && !errors.Is(err, net.ErrClosed) {
		return errnie.Err(errnie.Network, "swarm grpc mesh close failed", err)
	}

	return nil
}

/*
Address returns the bound listener address.
*/
func (transport *GRPCMeshTransport) Address() string {
	return transport.listener.Addr().String()
}

/*
Receive accepts one inbound gRPC publish request.
*/
func (transport *GRPCMeshTransport) Receive(
	ctx context.Context,
	envelope *MeshEnvelope,
) (*MeshAck, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if envelope == nil {
		return nil, errnie.Err(errnie.Validation, "swarm grpc mesh envelope is required", nil)
	}

	if transport.receive == nil {
		return nil, errnie.Err(errnie.Validation, "swarm grpc mesh receiver is required", nil)
	}

	if err := transport.receive(*envelope); err != nil {
		return nil, err
	}

	return &MeshAck{Accepted: true}, nil
}

func (transport *GRPCMeshTransport) serve() {
	err := transport.server.Serve(transport.listener)

	if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		errnie.Error(
			errnie.Err(errnie.Network, "swarm grpc mesh serve failed", err),
		)
	}
}

func (transport *GRPCMeshTransport) stopWhenDone(ctx context.Context) {
	<-ctx.Done()

	if err := transport.Close(); err != nil {
		errnie.Error(err)
	}
}
