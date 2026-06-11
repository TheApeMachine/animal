package editor

import (
	"github.com/theapemachine/animal/ai/tool/editor/agent"
	"github.com/theapemachine/animal/lease"
)

func principalFromAccess(access agent.Access) lease.Principal {
	return lease.Principal{
		ActorID:         access.ID,
		ReadOnly:        access.ReadOnly,
		AllowedPrefixes: access.LeasePrefixes,
		RequireLease:    access.RequireLease,
	}
}
