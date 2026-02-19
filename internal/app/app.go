package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"document-mdp/internal/config"
	"document-mdp/internal/db"
	"document-mdp/internal/embedding"
	"document-mdp/internal/httpapi"
	"document-mdp/internal/service/auth"
	"document-mdp/internal/service/search"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	h, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		_ = h.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.Migrate(ctx, h); err != nil {
		return err
	}

	emb, err := embedding.NewOpenAIProvider(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingModel)
	if err != nil {
		return err
	}

	authSvc := auth.NewService(h.Ent, cfg)
	searchSvc := search.NewService(h.Ent, h.SQL)

	r := httpapi.NewRouter(httpapi.Dependencies{
		Ent:       h.Ent,
		SQL:       h.SQL,
		Cfg:       cfg,
		Embedding: emb,
		Auth:      authSvc,
		Search:    searchSvc,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
