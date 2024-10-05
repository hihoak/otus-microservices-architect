package config

import "os"

type Config struct {
	Port string

	PostgresDSN string
}

var Cfg *Config

func init() {
	Cfg = &Config{
		Port:        os.Getenv("APP_PORT"),
		PostgresDSN: os.Getenv("APP_POSTGRES_DSN"),
	}
}
