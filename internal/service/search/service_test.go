package search

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

func TestService_Search_BuildsQueryAndScansRows(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	svc := &Service{ent: nil, sql: db}

	uid := uuid.New()
	qvec := pgvector.NewVector(make([]float32, 1536))
	year := 2024
	depth1 := "policy"
	depth2 := "security"
	alias := "ISMS"
	groupIDs := []string{uuid.New().String()}
	limit := 5

	// We don't assert the full SQL string (it changes easily), but we ensure
	// it contains key clauses: join embeddings, join alias, acl filtering and limit.
	// sqlmock matches entire SQL by default; use a regex that matches key parts.
	mock.ExpectQuery(`(?s).*FROM documents d.*JOIN document_embeddings de.*JOIN document_aliases al.*LEFT JOIN document_acls acl.*LIMIT.*`).
		WithArgs(sqlmock.AnyArg(), uid, year, depth1, depth2, "%"+alias+"%", sqlmock.AnyArg(), limit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "year", "depth1", "depth2", "title", "score"}).
			AddRow(uuid.New(), 2024, "policy", "security", "ISMS guide", 0.99))

	items, err := svc.Search(context.Background(), uid.String(), groupIDs, qvec, &year, &depth1, &depth2, &alias, limit)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("ExpectationsWereMet: %v", err)
	}
}
