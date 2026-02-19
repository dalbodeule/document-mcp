package embedding

import (
	"context"
	"fmt"

	"github.com/pgvector/pgvector-go"
	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	model  openai.EmbeddingModel
	dims   int
}

func NewOpenAIProvider(apiKey string, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is empty")
	}
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
		model:  openai.EmbeddingModel(model),
		dims:   1536,
	}, nil
}

func (p *OpenAIProvider) Embed(ctx context.Context, input string) (Result, error) {
	resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input:      input,
		Model:      p.model,
		Dimensions: p.dims,
	})
	if err != nil {
		return Result{}, fmt.Errorf("openai embeddings: %w", err)
	}
	if len(resp.Data) == 0 {
		return Result{}, fmt.Errorf("openai embeddings: empty response")
	}
	vec := resp.Data[0].Embedding
	if len(vec) == 0 {
		return Result{}, fmt.Errorf("openai embeddings: empty vector")
	}
	return Result{
		Model:  string(resp.Model),
		Dims:   len(vec),
		Vector: pgvector.NewVector(vec),
	}, nil
}
