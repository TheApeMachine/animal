package conversation

import (
	"context"

	"github.com/theapemachine/animal/ai/provider"
	"github.com/theapemachine/errnie"
)

func collectStreamReply(
	ctx context.Context,
	openai *provider.OpenAI,
	system string,
	agentCtx *provider.Context,
) (string, error) {
	err := openai.Stream(system, agentCtx, provider.NewParams())

	if err != nil {
		return "", errnie.Err(
			errnie.IO,
			"collect stream reply failed",
			err,
		)
	}

	return "", errnie.Error(err)
}
