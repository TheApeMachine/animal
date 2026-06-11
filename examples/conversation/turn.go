package conversation

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/theapemachine/animal/swarm"
)

/*
Turn is one conversational utterance in the plenary salon.
*/
type Turn struct {
	ActorID   string
	ActorName string
	Role      string
	Content   string
	At        int64
}

type turnPayload struct {
	Content string `json:"content"`
}

type stancePayload struct {
	Themes []string `json:"themes"`
}

var stanceLinePattern = regexp.MustCompile(`(?im)^STANCE:\s*(.+)\s*$`)

/*
PublishTurn gossips a spoken line to the salon.
*/
func PublishTurn(participant *swarm.Participant, content string) error {
	if content == "" {
		return fmt.Errorf("conversation: turn content is required")
	}

	raw, err := json.Marshal(turnPayload{Content: content})
	if err != nil {
		return fmt.Errorf("conversation: encode turn: %w", err)
	}

	return participant.Announce(TopicTurn, string(raw))
}

/*
PublishStance gossips thematic tags that summarize a speaker's current position.
*/
func PublishStance(participant *swarm.Participant, themes []string) error {
	if len(themes) == 0 {
		return fmt.Errorf("conversation: stance themes are required")
	}

	raw, err := json.Marshal(stancePayload{Themes: themes})
	if err != nil {
		return fmt.Errorf("conversation: encode stance: %w", err)
	}

	return participant.Announce(TopicStance, string(raw))
}

/*
ParseTurn decodes a gossip announce record into a Turn when applicable.
*/
func ParseTurn(record swarm.AnnounceRecord) (Turn, error) {
	if record.Topic != TopicTurn {
		return Turn{}, fmt.Errorf("conversation: record is not a turn")
	}

	var payload turnPayload
	if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
		return Turn{}, fmt.Errorf("conversation: decode turn: %w", err)
	}

	if payload.Content == "" {
		return Turn{}, fmt.Errorf("conversation: turn content is required")
	}

	return Turn{
		ActorID:   record.ActorID,
		ActorName: record.ActorName,
		Role:      record.Role,
		Content:   payload.Content,
		At:        record.At,
	}, nil
}

/*
ParseStance decodes thematic tags from a stance announce record.
*/
func ParseStance(record swarm.AnnounceRecord) ([]string, error) {
	if record.Topic != TopicStance {
		return nil, fmt.Errorf("conversation: record is not a stance")
	}

	var payload stancePayload
	if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
		return nil, fmt.Errorf("conversation: decode stance: %w", err)
	}

	if len(payload.Themes) == 0 {
		return nil, fmt.Errorf("conversation: stance themes are required")
	}

	return normalizeThemes(payload.Themes), nil
}

/*
SplitReplyAndStance separates spoken content from an optional trailing STANCE line.
*/
func SplitReplyAndStance(reply string) (string, []string) {
	trimmed := strings.TrimSpace(reply)
	match := stanceLinePattern.FindStringSubmatch(trimmed)

	if len(match) < 2 {
		return trimmed, nil
	}

	spoken := strings.TrimSpace(stanceLinePattern.ReplaceAllString(trimmed, ""))
	themes := normalizeThemes(strings.Split(match[1], ","))

	return spoken, themes
}

/*
FormatTranscript renders recent salon turns for LLM context.
*/
func FormatTranscript(turns []Turn, limit int) string {
	selected := make([]Turn, 0, limit)

	for index := len(turns) - 1; index >= 0 && len(selected) < limit; index-- {
		selected = append(selected, turns[index])
	}

	if len(selected) == 0 {
		return "The salon has no prior turns."
	}

	out := "Recent salon discussion:\n"

	for index := len(selected) - 1; index >= 0; index-- {
		turn := selected[index]
		stamp := time.Unix(0, turn.At).Format("15:04:05")
		out += fmt.Sprintf("[%s] %s (%s): %s\n", stamp, turn.ActorName, turn.Role, turn.Content)
	}

	return out
}

/*
ThemesFromContent derives stance tags from spoken text for clustering only.
*/
func ThemesFromContent(content string) []string {
	lower := strings.ToLower(content)
	lexicon := map[string][]string{
		"governance":  {"govern", "oversight", "charter", "policy", "regulat", "framework"},
		"rights":      {"right", "dignity", "autonom", "moral"},
		"law":         {"law", "legal", "enforce", "jurisdiction"},
		"metrics":     {"metric", "measure", "data", "baseline", "evidence"},
		"testing":     {"test", "regression", "falsif", "verify"},
		"engineering": {"build", "implement", "prototype", "ship", "code"},
		"hierarchy":   {"hierarch", "authority", "command", "order"},
		"dialogue":    {"dialogue", "discuss", "conversation", "consensus"},
		"risk":        {"risk", "harm", "failure", "safe"},
	}

	themes := make([]string, 0)

	for theme, keywords := range lexicon {
		for _, keyword := range keywords {
			if !strings.Contains(lower, keyword) {
				continue
			}

			themes = append(themes, theme)
			break
		}
	}

	if len(themes) == 0 {
		return []string{"general"}
	}

	return normalizeThemes(themes)
}

func normalizeThemes(raw []string) []string {
	themes := make([]string, 0, len(raw))

	for _, item := range raw {
		theme := strings.ToLower(strings.TrimSpace(item))
		if theme == "" {
			continue
		}

		themes = append(themes, theme)
	}

	return themes
}
