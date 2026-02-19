package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env string

	HTTPAddr string

	DatabaseURL string

	JWTAccessSecret   string
	JWTRefreshSecret  string
	AccessTTLSeconds  int
	RefreshTTLSeconds int

	OpenAIAPIKey         string
	OpenAIEmbeddingModel string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	c := Config{
		Env:                  getenv("APP_ENV", "dev"),
		HTTPAddr:             getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		JWTAccessSecret:      os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:     os.Getenv("JWT_REFRESH_SECRET"),
		AccessTTLSeconds:     getenvInt("JWT_ACCESS_TTL_SECONDS", 900),
		RefreshTTLSeconds:    getenvInt("JWT_REFRESH_TTL_SECONDS", 60*60*24*14),
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		OpenAIEmbeddingModel: getenv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTAccessSecret == "" || c.JWTRefreshSecret == "" {
		return Config{}, fmt.Errorf("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET are required")
	}
	if c.OpenAIAPIKey == "" {
		return Config{}, fmt.Errorf("OPENAI_API_KEY is required")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}
