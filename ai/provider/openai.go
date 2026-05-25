package provider

import (
	"context"
	"os"

	openaiapi "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/theapemachine/errnie"
	"github.com/theapemachine/qpool"
)

type OpenAI struct {
	ctx      context.Context
	cancel   context.CancelFunc
	err      error
	pool     *qpool.Q
	endpoint string
	apiKey   string
	model    string
	client   openaiapi.Client
}

func NewOpenAI(ctx context.Context, pool *qpool.Q) (*OpenAI, error) {
	ctx, cancel := context.WithCancel(ctx)

	openai := &OpenAI{
		ctx:      ctx,
		cancel:   cancel,
		pool:     pool,
		err:      nil,
		endpoint: "http://localhost:1234/v1",
		apiKey:   os.Getenv("OPENAI_API_KEY"),
		model:    "openai/gpt-oss-20b",
	}

	openai.client = openaiapi.NewClient(
		option.WithBaseURL(openai.endpoint),
		option.WithAPIKey(openai.apiKey),
	)

	return openai, errnie.Require(map[string]any{
		"ctx": openai.ctx,
	})
}

func (openai OpenAI) Stream(agentCtx *Context, returnChannel chan *qpool.QValue[any]) {
	messages := make([]responses.EasyInputMessage, 0)

	for _, message := range agentCtx.Messages {
		messages = append(messages, responses.EasyInputMessage{
			Content: responses.EasyInputMessageContentUnion{
				OfString: message.Content,
			},
			Role: responses.EasyInputMessageRole(message.Role),
		})
	}

	stream := openai.client.Responses.NewStreaming(openai.ctx, responses.ResponseNewParams{
		Model: openaiapi.ChatModelGPT5_2,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openaiapi.String("Write a haiku about programming"),
		},
	})

	for stream.Next() {
		event := stream.Current()
		print(event.Delta)
	}

	if stream.Err() != nil {
		panic(stream.Err())
	}
}
