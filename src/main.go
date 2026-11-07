package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("src/dbpass.env"); err != nil {
		log.Fatal("Error loading environment file:", err)
	}
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		log.Fatal("DATABASE_URL is missing from environment file")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("Unable to create connection pool:", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Could not connect to Supabase:", err)
	}

	fmt.Println("Successfully connected to Supabase!")
	rand.Seed(time.Now().UnixNano())
	fmt.Println("Welcome to the Planetary Game!")
}
