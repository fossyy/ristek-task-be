package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"ristek-task-be/internal/db/sqlc/repository"
	"ristek-task-be/internal/jwt"
	"ristek-task-be/internal/server"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if _, err := os.Stat(".env"); err == nil {
		if err = godotenv.Load(".env"); err != nil {
			log.Printf("Warning: Failed to load .env file: %s", err)
		}
	}

	errChan := make(chan error, 1)
	signalChan := make(chan os.Signal, 1)

	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	addr := "localhost"
	port := uint16(8080)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	connect, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		panic(err)
	}
	defer connect.Close()
	repo := repository.New(connect)
	j := jwt.New("yomama")
	go func() {
		s := server.New(addr, port, repo, j)
		err = s.Start()
		errChan <- err
	}()

	log.Printf("Server is running on %s:%d", addr, port)

	select {
	case err = <-errChan:
		log.Fatalf("service error: %w", err)
	case sig := <-signalChan:
		log.Printf("Received signal %s, initiating graceful shutdown", sig)
	}
}
