package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	Env                    string
	DatabasePath           string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	ResendAPIKey           string
	JWTSecret              string
}

func Load() *Config {

	_ = godotenv.Load()

	cfg := &Config{
		Port:                   getEnvOrDefault("PORT", "8080"),
		Env:                    getEnvOrDefault("ENV", "development"),
		DatabasePath:           getEnvOrDefault("DATABASE_PATH", "pfe.db"),
		SupabaseURL:            mustGetEnv("SUPABASE_URL"),
		SupabaseServiceRoleKey: mustGetEnv("SUPABASE_SERVICE_ROLE_KEY"),
		ResendAPIKey:           mustGetEnv("RESEND_API_KEY"),
		JWTSecret:              getEnvOrDefault("JWT_SECRET", "dev-secret"),
	}

	cfg.validate()
	return cfg
}

func (c *Config) validate() {
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		panic(fmt.Sprintf("ENV invalide : %s (attendu : development | production | test)", c.Env))
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("Variable d'environnement obligatoire manquante : %s", key))
	}
	return val
}

func getEnvOrDefault(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
