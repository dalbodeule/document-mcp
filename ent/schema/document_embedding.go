package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// DocumentEmbedding stores the vector representation for search.
type DocumentEmbedding struct {
	ent.Schema
}

func (DocumentEmbedding) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("model").NotEmpty().MaxLen(100),
		field.Int("dims"),
		field.Other("embedding", pgvector.Vector{}).
			SchemaType(map[string]string{dialect.Postgres: "vector(1536)"}),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.UUID("document_id", uuid.UUID{}),
	}
}

func (DocumentEmbedding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("document", Document.Type).Ref("embedding").Field("document_id").Unique().Required(),
	}
}

func (DocumentEmbedding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("document_id").Unique(),
		// NOTE: vector index (ivfflat/hnsw) should be created via SQL migration.
	}
}
