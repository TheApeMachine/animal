package conversation

import (
	"context"
	"time"

	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/qpool"
)

func collectStreamReply(
	ctx context.Context,
	openai *provider.OpenAI,
	system string,
	agentCtx *provider.Context,
) (string, error) {
	broadcast, err := qpool.NewBroadcastGroup(ctx, "conversation-stream", 64*time.Second)
	if err != nil {
		return "", err
	}

	err = openai.Stream(system, agentCtx, broadcast, provider.NewParams())
	if err != nil {
		return "", err
	}

	return "", nil
}
