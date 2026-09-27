// Package config reads the settings every bos API needs to start: its name, version and
// environment, the port, the database and cache addresses, the auth secret, the public base
// URL and the CORS allowlist.
//
// It is the loader of a running API, moved here rather than rewritten. What was hard-coded
// there (the product's variable prefix and its default name, port and addresses) is now
// passed in by the consumer; the differences from the original are recorded in the
// milestone plan.
package config

import (
	"os"
	"strings"
)

// Config holds application configuration.
type Config struct {
	AppName     string
	AppVersion  string
	Environment string
	Port        string
	DatabaseURL string

	// Auth config
	JWTSecret string
	BaseURL   string

	// Redis — used for rate limiting (C2) and session/JTI cache (H3).
	// Empty URL means no Redis: rate limiting falls back to in-memory and
	// logout cannot revoke access JWTs until they expire naturally.
	RedisURL string

	// CORSOrigins is the CORS allowlist, read with the other settings so that a
	// forbidden value stops the process at startup (see CorsOrigins).
	CORSOrigins []string
}

// Defaults are the values a consumer starts from. An empty field falls back to the
// framework's own default (version "0.1.0", environment "development", port "8080",
// CORS "http://localhost:*"; no name, database, base URL or cache).
type Defaults struct {
	AppName     string
	AppVersion  string
	Environment string
	Port        string
	DatabaseURL string
	BaseURL     string
	CORSOrigins []string
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func orDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

// envName scopes a product-named variable: with prefix "ACME" the name CORS_ORIGINS
// becomes ACME_CORS_ORIGINS; with no prefix it stays CORS_ORIGINS.
func envName(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "_" + name
}

// CorsOrigins returns the CORS allowlist. L5: NEVER returns "*" — credentialed
// CORS with wildcard origin is forbidden by browsers and would expose us to CSRF
// if proxied. Panics if <prefix>_CORS_ORIGINS contains "*" (fail fast at
// startup rather than at first request).
func CorsOrigins(prefix string, fallback []string) []string {
	if origins := os.Getenv(envName(prefix, "CORS_ORIGINS")); origins != "" {
		parts := strings.Split(origins, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
			if parts[i] == "*" {
				panic("L5: CORS allowlist must not contain '*' with AllowCredentials=true")
			}
		}
		return parts
	}
	if len(fallback) == 0 {
		return []string{"http://localhost:*"}
	}
	return fallback
}

// Load reads the configuration from the environment. The unprefixed names (APP_NAME,
// APP_VERSION, ENVIRONMENT, PORT, DATABASE_URL, JWT_SECRET, BASE_URL, REDIS_URL) are
// the same for every app; prefix scopes the product-named ones (today CORS_ORIGINS).
func Load(prefix string, d Defaults) Config {
	return Config{
		AppName:     getEnv("APP_NAME", d.AppName),
		AppVersion:  getEnv("APP_VERSION", orDefault(d.AppVersion, "0.1.0")),
		Environment: getEnv("ENVIRONMENT", orDefault(d.Environment, "development")),
		Port:        getEnv("PORT", orDefault(d.Port, "8080")),
		DatabaseURL: getEnv("DATABASE_URL", d.DatabaseURL),
		JWTSecret:   getEnv("JWT_SECRET", ""),
		BaseURL:     getEnv("BASE_URL", d.BaseURL),
		RedisURL:    getEnv("REDIS_URL", ""),
		CORSOrigins: CorsOrigins(prefix, d.CORSOrigins),
	}
}
