package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"log"
)

type Config struct {
	AppEnv      string `env:"APP_ENV" envDefault:"local"`
	AppPort     string `env:"APP_PORT" envDefault:"18080"`
	DatabaseURL string `env:"DATABASE_URL,required"`
}

func MustLoad() Config {
	_ = godotenv.Load()

	cfg, err := env.ParseAs[Config]()
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}

	return cfg
}
