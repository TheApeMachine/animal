package swarm

import (
	"cmp"
	"slices"
)

/*
Claims returns active lease-prefix claims in deterministic order.
*/
func (view *View) Claims() []ClaimRecord {
	snapshot := view.state.Load()
	claims := make([]ClaimRecord, 0, len(snapshot.claims))

	for prefix, record := range snapshot.claims {
		claims = append(claims, ClaimRecord{
			Prefix:    prefix,
			ActorID:   record.actorID,
			ActorName: record.actorName,
			Role:      record.role,
			At:        record.at,
		})
	}

	slices.SortFunc(claims, func(firstClaim, secondClaim ClaimRecord) int {
		return cmp.Compare(firstClaim.Prefix, secondClaim.Prefix)
	})

	return claims
}

/*
Statuses returns the latest actor statuses in deterministic order.
*/
func (view *View) Statuses() []StatusRecord {
	snapshot := view.state.Load()
	statuses := make([]StatusRecord, 0, len(snapshot.statuses))

	for actorID, record := range snapshot.statuses {
		statuses = append(statuses, StatusRecord{
			ActorID:   actorID,
			ActorName: record.actorName,
			Role:      record.role,
			State:     record.state,
			At:        record.at,
		})
	}

	slices.SortFunc(statuses, func(firstStatus, secondStatus StatusRecord) int {
		return cmp.Compare(firstStatus.ActorID, secondStatus.ActorID)
	})

	return statuses
}

/*
AllTaskClaims returns every task claim in deterministic confirmation order.
*/
func (view *View) AllTaskClaims() []TaskClaim {
	snapshot := view.state.Load()
	taskClaims := make([]TaskClaim, 0)

	for _, claims := range snapshot.taskClaims {
		for _, claim := range claims {
			taskClaims = append(taskClaims, claim)
		}
	}

	slices.SortFunc(taskClaims, compareDashboardTaskClaims)

	return taskClaims
}

func compareDashboardTaskClaims(firstClaim, secondClaim TaskClaim) int {
	if diff := cmp.Compare(firstClaim.TaskID, secondClaim.TaskID); diff != 0 {
		return diff
	}

	return compareTaskClaims(firstClaim, secondClaim)
}
