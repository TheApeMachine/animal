package swarm

import "fmt"

func (detector *ContentionDetector) detectStarvation(
	contentions []Contention,
) []ContentionIssue {
	groups := make(map[string][]Contention)

	for _, contention := range contentions {
		key := contention.ActorID + "\x00" + contention.Prefix
		groups[key] = append(groups[key], contention)
	}

	issues := make([]ContentionIssue, 0)

	for _, group := range groups {
		if len(group) < detector.starvationThreshold {
			continue
		}

		issues = append(issues, detector.starvationIssue(group))
	}

	return issues
}

func (detector *ContentionDetector) starvationIssue(
	contentions []Contention,
) ContentionIssue {
	firstContention := contentions[0]
	lastContention := contentions[len(contentions)-1]

	return ContentionIssue{
		Kind:         ContentionIssueStarvation,
		ActorID:      firstContention.ActorID,
		Prefix:       firstContention.Prefix,
		HolderID:     lastContention.HolderID,
		HolderPrefix: lastContention.HolderPrefix,
		Attempts:     len(contentions),
		Since:        firstContention.At,
		Until:        lastContention.At,
		Summary: fmt.Sprintf(
			"actor %q is starved on prefix %q after %d attempts",
			firstContention.ActorID,
			firstContention.Prefix,
			len(contentions),
		),
	}
}
