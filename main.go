package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"os"
	"os/signal"
	"ristek-task-be/internal/config"
	"ristek-task-be/internal/db/sqlc/repository"
	"ristek-task-be/internal/jwt"
	"ristek-task-be/internal/server"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed internal/db/sqlc/migrations/*.sql
var migrationFiles embed.FS

// @title Ristek Task API
// @version 1.0
// @description REST API for Ristek Task Backend
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

// @host localhost:8080
// @BasePath /
func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	conf, err := config.MustLoad()
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	errChan := make(chan error, 1)
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	connect, err := pgxpool.New(ctx, conf.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
		return
	}
	defer connect.Close()
	repo := repository.New(connect)

	d, err := iofs.New(migrationFiles, "internal/db/sqlc/migrations")
	if err != nil {
		log.Fatalf("failed to create migration source: %s", err)
		return
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, conf.DatabaseURL())
	if err != nil {
		log.Fatalf("failed to initialize migrations: %s", err)
		return
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(err)
		return
	}

	j := jwt.New(conf.JwtSecret())
	go func() {
		s := server.New(conf.Addr(), conf.Port(), repo, j)
		err = s.Start()
		errChan <- err
	}()

	log.Printf("Server is running on %s:%s", conf.Addr(), conf.Port())

	select {
	case err = <-errChan:
		log.Fatalf("service error: %w", err)
	case sig := <-signalChan:
		log.Printf("Received signal %s, initiating graceful shutdown", sig)
	}
}
