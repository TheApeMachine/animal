package lease

import "github.com/theapemachine/animal/actor"

/*
Principal carries actor identity and lease policy for a single operation.
*/
type Principal struct {
	ActorID         string
	ReadOnly        bool
	AllowedPrefixes []string
	RequireLease    bool
}

/*
PrincipalFromIdentity builds a principal from a registered actor identity.
*/
func PrincipalFromIdentity(identity *actor.Identity) Principal {
	if identity == nil {
		return Principal{}
	}

	return Principal{ActorID: identity.ID}
}
