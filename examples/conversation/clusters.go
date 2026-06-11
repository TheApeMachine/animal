package conversation

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

/*
OpinionCluster is an emergent alignment group derived from shared stance themes.
*/
type OpinionCluster struct {
	ID      int
	Themes  []string
	Members []string
}

/*
ComputeClusters groups actors whose distinctive stance themes overlap enough.
Universal themes shared by most of the panel are ignored to avoid megaclusters.
*/
func ComputeClusters(
	registry *SalonRegistry,
	names map[string]string,
	minOverlap float64,
) []OpinionCluster {
	actorIDs := sortedActorIDs(registry, names)
	if len(actorIDs) < 2 {
		return nil
	}

	universal := universalThemes(registry, actorIDs)
	distinctive := make(map[string]map[string]struct{}, len(actorIDs))

	for _, actorID := range actorIDs {
		distinctive[actorID] = distinctiveThemes(registry.ThemeSet(actorID), universal)
	}

	parent := make(map[string]string, len(actorIDs))
	rank := make(map[string]int, len(actorIDs))

	for _, actorID := range actorIDs {
		parent[actorID] = actorID
		rank[actorID] = 0
	}

	for left := 0; left < len(actorIDs); left++ {
		for right := left + 1; right < len(actorIDs); right++ {
			leftID := actorIDs[left]
			rightID := actorIDs[right]

			if themeOverlap(distinctive[leftID], distinctive[rightID]) < minOverlap {
				continue
			}

			unionSets(parent, rank, leftID, rightID)
		}
	}

	buckets := make(map[string][]string)

	for _, actorID := range actorIDs {
		root := findSet(parent, actorID)
		buckets[root] = append(buckets[root], actorID)
	}

	clusters := make([]OpinionCluster, 0)

	for _, members := range buckets {
		if len(members) < 2 {
			continue
		}

		sort.Strings(members)
		themes := sharedDistinctiveThemes(distinctive, members)
		if len(themes) == 0 {
			continue
		}

		label := clusterLabel(names, members, themes)

		clusters = append(clusters, OpinionCluster{
			ID:      len(clusters) + 1,
			Themes:  themes,
			Members: label,
		})
	}

	sort.Slice(clusters, func(left, right int) bool {
		return clusters[left].Members[0] < clusters[right].Members[0]
	})

	return clusters
}

/*
FormatClusters renders emergent alignment groups for the console.
*/
func FormatClusters(clusters []OpinionCluster) string {
	if len(clusters) == 0 {
		return "opinion clusters: (none yet — need overlapping stances from at least two speakers)"
	}

	lines := make([]string, 0, len(clusters)+1)
	lines = append(lines, "opinion clusters (emergent from shared themes):")

	for _, cluster := range clusters {
		lines = append(lines, fmt.Sprintf(
			"  #%d themes=%s members=%s",
			cluster.ID,
			strings.Join(cluster.Themes, ", "),
			strings.Join(cluster.Members, ", "),
		))
	}

	return strings.Join(lines, "\n")
}

/*
AlignedPeers lists other actor names in the same emergent cluster.
*/
func AlignedPeers(
	registry *SalonRegistry,
	names map[string]string,
	actorID string,
	minOverlap float64,
) []string {
	clusters := ComputeClusters(registry, names, minOverlap)
	selfName := names[actorID]

	peers := make([]string, 0)

	for _, cluster := range clusters {
		inCluster := false

		for _, member := range cluster.Members {
			if member == selfName {
				inCluster = true
				break
			}
		}

		if !inCluster {
			continue
		}

		for _, member := range cluster.Members {
			if member == selfName {
				continue
			}

			peers = append(peers, member)
		}
	}

	sort.Strings(peers)

	return peers
}

func sortedActorIDs(registry *SalonRegistry, names map[string]string) []string {
	seen := make(map[string]struct{}, len(names))

	for _, turn := range registry.Turns() {
		seen[turn.ActorID] = struct{}{}
	}

	for actorID := range names {
		if len(registry.ThemeSet(actorID)) > 0 {
			seen[actorID] = struct{}{}
		}
	}

	actorIDs := make([]string, 0, len(seen))
	for actorID := range seen {
		actorIDs = append(actorIDs, actorID)
	}

	sort.Strings(actorIDs)

	return actorIDs
}

func sharedDistinctiveThemes(
	distinctive map[string]map[string]struct{},
	members []string,
) []string {
	if len(members) == 0 {
		return nil
	}

	shared := maps.Clone(distinctive[members[0]])

	for _, actorID := range members[1:] {
		themes := distinctive[actorID]

		for theme := range shared {
			if _, ok := themes[theme]; ok {
				continue
			}

			delete(shared, theme)
		}
	}

	out := make([]string, 0, len(shared))
	for theme := range shared {
		out = append(out, theme)
	}

	sort.Strings(out)

	return out
}

func universalThemes(registry *SalonRegistry, actorIDs []string) map[string]struct{} {
	counts := make(map[string]int)

	for _, actorID := range actorIDs {
		for theme := range registry.ThemeSet(actorID) {
			if theme == "general" {
				continue
			}

			counts[theme]++
		}
	}

	universal := make(map[string]struct{})

	for theme, count := range counts {
		if count == len(actorIDs) {
			universal[theme] = struct{}{}
		}
	}

	return universal
}

func distinctiveThemes(full, universal map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{})

	for theme := range full {
		if theme == "general" {
			continue
		}

		if _, ok := universal[theme]; ok {
			continue
		}

		out[theme] = struct{}{}
	}

	return out
}

func sharedThemes(registry *SalonRegistry, members []string) []string {
	if len(members) == 0 {
		return nil
	}

	shared := maps.Clone(registry.ThemeSet(members[0]))

	for _, actorID := range members[1:] {
		themes := registry.ThemeSet(actorID)

		for theme := range shared {
			if _, ok := themes[theme]; ok {
				continue
			}

			delete(shared, theme)
		}
	}

	out := make([]string, 0, len(shared))
	for theme := range shared {
		out = append(out, theme)
	}

	sort.Strings(out)

	return out
}

func clusterLabel(names map[string]string, members []string, themes []string) []string {
	labels := make([]string, 0, len(members))

	for _, actorID := range members {
		name := names[actorID]
		if name == "" {
			name = actorID
		}

		labels = append(labels, name)
	}

	return labels
}

func themeOverlap(left, right map[string]struct{}) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}

	intersection := 0

	for theme := range left {
		if _, ok := right[theme]; ok {
			intersection++
		}
	}

	union := len(left) + len(right) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func findSet(parent map[string]string, actorID string) string {
	if parent[actorID] != actorID {
		parent[actorID] = findSet(parent, parent[actorID])
	}

	return parent[actorID]
}

func unionSets(parent map[string]string, rank map[string]int, leftID, rightID string) {
	leftRoot := findSet(parent, leftID)
	rightRoot := findSet(parent, rightID)

	if leftRoot == rightRoot {
		return
	}

	if rank[leftRoot] < rank[rightRoot] {
		parent[leftRoot] = rightRoot
		return
	}

	parent[rightRoot] = leftRoot

	if rank[leftRoot] != rank[rightRoot] {
		return
	}

	rank[leftRoot]++
}
