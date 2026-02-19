package db

import (
	"database/sql"
	"fmt"

	"document-mdp/ent"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Handle struct {
	SQL *sql.DB
	Ent *ent.Client
}

func Open(databaseURL string) (*Handle, error) {
	// Use pgx stdlib driver.
	sqldb, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}
	if err := sqldb.Ping(); err != nil {
		return nil, fmt.Errorf("sql ping: %w", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, sqldb)
	client := ent.NewClient(ent.Driver(drv))

	return &Handle{SQL: sqldb, Ent: client}, nil
}

func (h *Handle) Close() error {
	if h.Ent != nil {
		_ = h.Ent.Close()
	}
	if h.SQL != nil {
		return h.SQL.Close()
	}
	return nil
}
