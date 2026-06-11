package swarm

/*
Kind identifies a gossip rumor type propagated on the swarm mesh.
*/
type Kind string

const (
	KindAnnounce Kind = "announce"
	KindClaim    Kind = "claim"
	KindRelease  Kind = "release"
	KindStatus   Kind = "status"
)
