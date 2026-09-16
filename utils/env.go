package utils

import (
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type ConfigEnv struct {
	AllowedOrigins       []string `env:"ALLOWED_ORIGINS" envDefault:"http://localhost:5173,http://localhost:3000"`
	NumberOfAllowedUsers int      `env:"NUMBER_OF_ALLOWED_USERS" envDefault:"50"`
	RemoveExpiredSession int      `env:"REMOVE_EXPIRED_SESSION" envDefault:"30"`
	Port                 string   `env:"PORT" envDefault:"3000"`
}

func LoadConfig() (*ConfigEnv, error) {
	if os.Getenv("APP_ENV") != "PRODUCTION" {
		_ = godotenv.Load()
	}

	cfg := &ConfigEnv{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	for i, origin := range cfg.AllowedOrigins {
		cfg.AllowedOrigins[i] = strings.TrimSpace(origin)
	}

	return cfg, nil
}
