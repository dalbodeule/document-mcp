package embedding

import (
	"context"

	"github.com/pgvector/pgvector-go"
)

type Result struct {
	Model  string
	Dims   int
	Vector pgvector.Vector
}

type Provider interface {
	Embed(ctx context.Context, input string) (Result, error)
}
