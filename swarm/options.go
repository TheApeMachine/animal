package swarm

import "time"

/*
Options configures a swarm registry and its gossip mesh.
*/
type Options struct {
	MeshID    string
	GossipTTL time.Duration
	MeshTTL   time.Duration
	Buffer    int
	Transport MeshTransport
}
