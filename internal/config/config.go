package config

import (
	"os"
	"time"
)

type Config struct {
	Env            string
	HTTPAddr       string
	DatabaseURL    string
	JWTIssuer      string
	AccessTokenTTL time.Duration
}

func Load() Config {
	ttl, err := time.ParseDuration(get("ACCESS_TOKEN_TTL", "15m"))
	if err != nil || ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return Config{
		Env:            get("APP_ENV", "development"),
		HTTPAddr:       get("HTTP_ADDR", ":8080"),
		DatabaseURL:    get("DATABASE_URL", "postgres://doit:doit_dev_password@localhost:5432/doit?sslmode=disable"),
		JWTIssuer:      get("JWT_ISSUER", "doit-api"),
		AccessTokenTTL: ttl,
	}
}
func get(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
