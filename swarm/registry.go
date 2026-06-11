package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/theapemachine/animal/lease"
	"github.com/theapemachine/qpool"
)

/*
Registry owns the shared mesh and lease coordinator for a swarm run.
*/
type Registry struct {
	mesh        *Mesh
	coordinator *lease.Coordinator
	gossipTTL   time.Duration
	buffer      int
}

/*
NewRegistry wires shared swarm infrastructure from options and lease config.
*/
func NewRegistry(
	ctx context.Context,
	pool *qpool.Q[any],
	options Options,
	leaseOptions lease.Options,
) (*Registry, error) {
	mesh, err := NewMesh(ctx, pool, options)

	if err != nil {
		return nil, err
	}

	coordinator, err := lease.NewCoordinator(leaseOptions)

	if err != nil {
		return nil, err
	}

	return &Registry{
		mesh:        mesh,
		coordinator: coordinator,
		gossipTTL:   options.GossipTTL,
		buffer:      options.Buffer,
	}, nil
}

/*
Coordinator returns the shared lease registry for filesystem exclusivity.
*/
func (registry *Registry) Coordinator() *lease.Coordinator {
	return registry.coordinator
}

/*
Mesh returns the shared gossip transport.
*/
func (registry *Registry) Mesh() *Mesh {
	return registry.mesh
}

/*
NewParticipant registers an actor on the mesh with a private local view.
*/
func (registry *Registry) NewParticipant(
	actorID, actorName, role string,
	claimPrefixes []string,
) (*Participant, error) {
	if actorID == "" {
		return nil, fmt.Errorf("swarm: participant actor ID is required")
	}

	view, err := NewView(registry.gossipTTL)

	if err != nil {
		return nil, err
	}

	subscriber, err := registry.mesh.Subscribe(actorID, registry.buffer)

	if err != nil {
		return nil, err
	}

	return &Participant{
		actorID:       actorID,
		actorName:     actorName,
		role:          role,
		claimPrefixes: append([]string(nil), claimPrefixes...),
		view:          view,
		mesh:          registry.mesh,
		coordinator:   registry.coordinator,
		subscriber:    subscriber,
	}, nil
}
