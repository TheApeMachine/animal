package swarm

import "fmt"

func (detector *ContentionDetector) detectDeadlocks(
	contentions []Contention,
) []ContentionIssue {
	latest := detector.latestContentionByActor(contentions)
	issues := make([]ContentionIssue, 0)
	seen := make(map[string]bool)

	for actorID, contention := range latest {
		other, ok := latest[contention.HolderID]

		if !ok {
			continue
		}

		if other.HolderID != actorID {
			continue
		}

		key := contentionDeadlockKey(actorID, contention.HolderID)
		if seen[key] {
			continue
		}

		seen[key] = true
		issues = append(issues, detector.deadlockIssue(contention, other))
	}

	return issues
}

func (detector *ContentionDetector) latestContentionByActor(
	contentions []Contention,
) map[string]Contention {
	latest := make(map[string]Contention)

	for _, contention := range contentions {
		if contention.HolderID == "" {
			continue
		}

		existing, ok := latest[contention.ActorID]
		if ok && existing.At >= contention.At {
			continue
		}

		latest[contention.ActorID] = contention
	}

	return latest
}

func (detector *ContentionDetector) deadlockIssue(
	firstContention Contention,
	secondContention Contention,
) ContentionIssue {
	return ContentionIssue{
		Kind:         ContentionIssueDeadlock,
		ActorID:      firstContention.ActorID,
		Prefix:       firstContention.Prefix,
		HolderID:     firstContention.HolderID,
		HolderPrefix: firstContention.HolderPrefix,
		Attempts:     2,
		Since:        firstContention.At,
		Until:        secondContention.At,
		Summary: fmt.Sprintf(
			"actors %q and %q are waiting on each other's prefixes",
			firstContention.ActorID,
			secondContention.ActorID,
		),
	}
}

func contentionDeadlockKey(firstActorID, secondActorID string) string {
	if firstActorID < secondActorID {
		return firstActorID + "\x00" + secondActorID
	}

	return secondActorID + "\x00" + firstActorID
}
