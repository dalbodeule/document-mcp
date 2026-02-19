package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"document-mdp/ent"
	"document-mdp/ent/document"
	"document-mdp/ent/documentalias"
	"document-mdp/ent/documentembedding"
	"document-mdp/internal/embedding"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

type Service struct {
	ent *ent.Client
	sql *sql.DB
}

func NewService(entClient *ent.Client, sqlDB *sql.DB) *Service {
	return &Service{ent: entClient, sql: sqlDB}
}

type SearchItem struct {
	ID     uuid.UUID `json:"id"`
	Year   int       `json:"year"`
	Depth1 string    `json:"depth1"`
	Depth2 string    `json:"depth2"`
	Title  string    `json:"title"`
	Score  float64   `json:"score"`
}

func (s *Service) Search(ctx context.Context, userID string, groupIDs []string, qvec pgvector.Vector, year *int, depth1, depth2, alias *string, limit int) ([]SearchItem, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user")
	}

	args := []any{qvec, uid}
	argn := 2

	where := []string{"1=1"}
	if year != nil {
		argn++
		where = append(where, fmt.Sprintf("d.year = $%d", argn))
		args = append(args, *year)
	}
	if depth1 != nil {
		argn++
		where = append(where, fmt.Sprintf("d.depth1 = $%d", argn))
		args = append(args, *depth1)
	}
	if depth2 != nil {
		argn++
		where = append(where, fmt.Sprintf("d.depth2 = $%d", argn))
		args = append(args, *depth2)
	}

	joinAlias := ""
	if alias != nil && strings.TrimSpace(*alias) != "" {
		joinAlias = "JOIN document_aliases al ON al.document_id = d.id"
		argn++
		where = append(where, fmt.Sprintf("al.alias ILIKE $%d", argn))
		args = append(args, "%"+strings.TrimSpace(*alias)+"%")
	}

	argn++
	gidsArgPos := argn
	args = append(args, pq.Array(pqUUIDTextArray(groupIDs)))

	argn++
	limitPos := argn
	args = append(args, limit)

	q := fmt.Sprintf(`
WITH ranked AS (
	SELECT
		d.id,
		d.year,
		d.depth1,
		d.depth2,
		d.title,
		d.created_by_user_id,
		(1 - (de.embedding <=> $1))::float8 AS score,
		bool_or(acl.effect = 'deny') AS has_deny,
		bool_or(acl.effect IN ('read','read_write')) AS has_allow
	FROM documents d
	JOIN document_embeddings de ON de.document_id = d.id
	%s
	LEFT JOIN document_acls acl
		ON acl.document_id = d.id
		AND acl.group_id = ANY($%d::uuid[])
	WHERE %s
	GROUP BY d.id, d.year, d.depth1, d.depth2, d.title, d.created_by_user_id, de.embedding
)
SELECT id, year, depth1, depth2, title, score
FROM ranked
WHERE
	(has_deny IS DISTINCT FROM true)
	AND (created_by_user_id = $2 OR (has_allow = true))
ORDER BY score DESC
LIMIT $%d;
`, joinAlias, gidsArgPos, strings.Join(where, " AND "), limitPos)

	rows, err := s.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SearchItem, 0, limit)
	for rows.Next() {
		var it SearchItem
		if err := rows.Scan(&it.ID, &it.Year, &it.Depth1, &it.Depth2, &it.Title, &it.Score); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// CreateDocumentWithEmbedding persists document, aliases and embedding in a single transaction.
func (s *Service) CreateDocumentWithEmbedding(
	ctx context.Context,
	createdBy uuid.UUID,
	year int, depth1, depth2, title, content string,
	aliases []string,
	emb embedding.Result,
) (*ent.Document, error) {
	tx, err := s.ent.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	docRow, err := tx.Document.Create().
		SetYear(year).
		SetDepth1(depth1).
		SetDepth2(depth2).
		SetTitle(title).
		SetContent(content).
		SetCreatedByUserID(createdBy).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	trimmed := make([]string, 0, len(aliases))
	seen := map[string]struct{}{}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		key := strings.ToLower(a)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		trimmed = append(trimmed, a)
	}
	if len(trimmed) > 0 {
		creates := make([]*ent.DocumentAliasCreate, 0, len(trimmed))
		for _, a := range trimmed {
			creates = append(creates, tx.DocumentAlias.Create().SetDocumentID(docRow.ID).SetAlias(a))
		}
		if err := tx.DocumentAlias.CreateBulk(creates...).Exec(ctx); err != nil {
			return nil, err
		}
	}

	// upsert embedding for document
	if err := tx.DocumentEmbedding.Create().
		SetDocumentID(docRow.ID).
		SetModel(emb.Model).
		SetDims(emb.Dims).
		SetEmbedding(emb.Vector).
		OnConflictColumns(documentembedding.FieldDocumentID).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return docRow, nil
}

func (s *Service) GetDocument(ctx context.Context, docID uuid.UUID) (*ent.Document, []string, error) {
	docRow, err := s.ent.Document.Query().Where(document.IDEQ(docID)).Only(ctx)
	if err != nil {
		return nil, nil, err
	}
	als, err := s.ent.DocumentAlias.Query().Where(documentalias.DocumentIDEQ(docID)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	aliases := make([]string, 0, len(als))
	for _, a := range als {
		aliases = append(aliases, a.Alias)
	}
	return docRow, aliases, nil
}

// pqUUIDTextArray passes UUID list as text[] and casts to uuid[] in SQL.
func pqUUIDTextArray(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, err := uuid.Parse(id); err == nil {
			out = append(out, id)
		}
	}
	return out
}
