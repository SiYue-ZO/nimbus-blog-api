package webapi

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/scc749/nimbus-blog-api/internal/repo"
)

type llmWebAPI struct {
	client openai.Client
	model  string
}

func NewLLMWebAPI(apiKey, baseURL, model string) repo.LLMWebAPI {
	c := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL))
	return &llmWebAPI{client: c, model: model}
}

func (l *llmWebAPI) Complete(ctx context.Context, system string, user string) (string, error) {
	res, err := l.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: l.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(system),
			openai.UserMessage(user),
		},
	})
	if err != nil {
		return "", fmt.Errorf("LLMWebAPI - Complete - client.Chat.Completions.New: %w", err)
	}
	if len(res.Choices) == 0 {
		return "", fmt.Errorf("LLMWebAPI - Complete - choices empty")
	}
	return res.Choices[0].Message.Content, nil
}
