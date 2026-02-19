package db

import (
	"context"
	"database/sql"
	"fmt"

	"document-mdp/ent"
)

// Migrate runs minimal DB initialization required for pgvector + ent schema.
// Order is important:
//  1. pgvector extension
//  2. ent schema create
//  3. vector index
func Migrate(ctx context.Context, h *Handle) error {
	if h == nil || h.SQL == nil || h.Ent == nil {
		return fmt.Errorf("db handle is nil")
	}
	if err := EnsurePgvector(ctx, h.SQL); err != nil {
		return err
	}
	if err := h.Ent.Schema.Create(ctx); err != nil {
		return fmt.Errorf("ent schema create: %w", err)
	}
	if err := EnsureEmbeddingIndex(ctx, h.SQL); err != nil {
		return err
	}
	return nil
}

// Compile-time check that we didn't accidentally drift from ent client type.
var _ *ent.Client

func EnsurePgvector(ctx context.Context, db ExecContext) error {
	_, err := db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS vector;`)
	if err != nil {
		return fmt.Errorf("create extension vector: %w", err)
	}
	return nil
}

type ExecContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func EnsureEmbeddingIndex(ctx context.Context, db ExecContext) error {
	// NOTE: Table/column names follow ent's default naming.
	// ivfflat requires ANALYZE after data load and a suitable lists parameter.
	_, err := db.ExecContext(ctx, `
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND indexname = 'document_embeddings_embedding_ivfflat_idx'
	) THEN
		CREATE INDEX document_embeddings_embedding_ivfflat_idx
		ON document_embeddings
		USING ivfflat (embedding vector_cosine_ops)
		WITH (lists = 100);
	END IF;
END$$;
`)
	if err != nil {
		return fmt.Errorf("create embedding index: %w", err)
	}
	return nil
}
