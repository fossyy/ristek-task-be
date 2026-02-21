package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config interface {
	Addr() string
	Port() string
	DatabaseURL() string
}
type config struct {
	addr        string
	port        string
	databaseURL string
}

func (c *config) Addr() string {
	return c.addr
}

func (c *config) Port() string {
	return c.port
}

func (c *config) DatabaseURL() string {
	return c.databaseURL
}

func parse() (*config, error) {
	domain := getenv("ADDRESS", "0.0.0.0")
	sshPort := getenv("PORT", "8080")
	databaseURL := getenv("DATABASE_URL", "")

	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	return &config{
		addr:        domain,
		port:        sshPort,
		databaseURL: databaseURL,
	}, nil
}

func loadEnvFile() error {
	if _, err := os.Stat(".env"); err == nil {
		return godotenv.Load(".env")
	}
	return nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func MustLoad() (Config, error) {
	if err := loadEnvFile(); err != nil {
		return nil, err
	}

	cfg, err := parse()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
