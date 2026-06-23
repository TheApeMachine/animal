package swarm

import (
	"slices"
	"strings"

	"github.com/theapemachine/errnie"
)

func cleanGRPCPeers(peers []string) ([]string, error) {
	peerAddresses := slices.Clone(peers)

	for _, peer := range peerAddresses {
		if strings.TrimSpace(peer) != "" {
			continue
		}

		return nil, errnie.Err(
			errnie.Validation,
			"swarm grpc mesh peer address is required",
			nil,
		)
	}

	return peerAddresses, nil
}
