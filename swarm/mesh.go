package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

/*
Mesh routes swarm rumors through a shared qpool broadcast group.
*/
type Mesh struct {
	ctx       context.Context
	cancel    context.CancelFunc
	err       error
	group     *qpool.BroadcastGroup
	gossipTTL time.Duration
}

/*
NewMesh attaches to or creates the broadcast group identified by meshID.
*/
func NewMesh(
	ctx context.Context,
	pool *qpool.Q[any],
	options Options,
) (*Mesh, error) {
	ctx, cancel := context.WithCancel(ctx)

	mesh := &Mesh{
		ctx:       ctx,
		cancel:    cancel,
		group:     pool.CreateBroadcastGroup(options.MeshID, options.MeshTTL),
		gossipTTL: options.GossipTTL,
	}

	return mesh, errnie.Require(map[string]any{
		"ctx":       mesh.ctx,
		"cancel":    mesh.cancel,
		"group":     mesh.group,
		"gossipTTL": mesh.gossipTTL,
	})
}

/*
Subscribe registers an actor on the mesh receive path.
*/
func (mesh *Mesh) Subscribe(actorID string, buffer int) (*qpool.BroadcastConsumer, error) {
	if actorID == "" {
		return nil, fmt.Errorf("swarm: subscriber actor ID is required")
	}

	if buffer <= 0 {
		return nil, fmt.Errorf("swarm: subscriber buffer is required")
	}

	subscriber := mesh.group.Subscribe(actorID, buffer)

	if subscriber == nil {
		return nil, fmt.Errorf("swarm: mesh subscribe failed for actor %q", actorID)
	}

	return subscriber, nil
}

/*
Publish sends a rumor to all mesh subscribers except the sender.
*/
func (mesh *Mesh) Publish(senderID string, rumor Rumor) error {
	if err := rumor.Validate(); err != nil {
		return err
	}

	qv, err := qpool.NewQValue[any](senderID, "", rumor, mesh.gossipTTL)

	if err != nil {
		return err
	}

	mesh.group.Send(qv)

	return nil
}

/*
GroupID returns the underlying broadcast group identifier.
*/
func (mesh *Mesh) GroupID() string {
	return mesh.group.ID
}
