package conversation

import (
	"fmt"
	"strings"

	"github.com/theapemachine/animal/ai/provider"
)

/*
LastTurn returns the most recent salon utterance when one exists.
*/
func LastTurn(turns []Turn) (Turn, bool) {
	if len(turns) == 0 {
		return Turn{}, false
	}

	return turns[len(turns)-1], true
}

/*
BuildReplyMessages returns chat history with a persistent moderator anchor and role separation.
Others appear as user lines; the speaking agent's own prior lines appear as assistant lines.
*/
func BuildReplyMessages(turns []Turn, openingQuestion, selfName string) []provider.Message {
	messages := make([]provider.Message, 0, len(turns)+1)

	anchor := strings.TrimSpace(openingQuestion)
	if anchor != "" {
		messages = append(messages, provider.Message{
			Role:    "user",
			Content: fmt.Sprintf("Moderator: %s", anchor),
		})
	}

	for _, turn := range turns {
		if turn.ActorName == selfName {
			messages = append(messages, provider.Message{
				Role:    "assistant",
				Content: turn.Content,
			})

			continue
		}

		messages = append(messages, provider.Message{
			Role:    "user",
			Content: fmt.Sprintf("%s: %s", turn.ActorName, turn.Content),
		})
	}

	return messages
}
