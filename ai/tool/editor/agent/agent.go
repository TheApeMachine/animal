package agent

import (
	"net/http"
	"strings"
)

/*
Access describes what an agent may do inside the workspace.
*/
type Access struct {
	ID            string
	ReadOnly      bool
	LeasePrefixes []string
	RequireLease  bool
}

/*
DefaultAccess is used when no agent headers are present for solo workflows.
*/
func DefaultAccess() Access {
	return Access{ID: "default"}
}

/*
ParseHeaders extracts agent access from MCP session headers.
*/
func ParseHeaders(header http.Header) Access {
	if header == nil {
		return DefaultAccess()
	}

	agentID := strings.TrimSpace(header.Get("X-Agent-ID"))
	if agentID == "" {
		return DefaultAccess()
	}

	access := Access{
		ID:           agentID,
		ReadOnly:     strings.EqualFold(header.Get("X-Agent-Read-Only"), "true"),
		RequireLease: strings.EqualFold(header.Get("X-Agent-Require-Lease"), "true"),
	}

	raw := header.Get("X-Agent-Lease-Prefixes")
	for part := range strings.SplitSeq(raw, ",") {
		prefix := strings.TrimSpace(part)
		if prefix != "" {
			access.LeasePrefixes = append(access.LeasePrefixes, prefix)
		}
	}

	return access
}
