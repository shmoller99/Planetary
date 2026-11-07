package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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
	var dbTime time.Time

	err = pool.QueryRow(ctx, "SELECT NOW()").Scan(&dbTime)
	if err != nil {
		log.Fatal("Query failed:", err)
	}

	fmt.Println("Database time:", dbTime)
	username := "alice"
	planetName := "Blue Haven"

	_, err = pool.Exec(ctx,
		"INSERT INTO users (username, planet_name) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING",
		username,
		planetName,
	)

	if err != nil {
		log.Fatal("Failed to insert user:", err)
	}

	fmt.Println("Successfully created user:", username)
	var retrievedUsername string
	var retrievedPlanet string

	err = pool.QueryRow(ctx,
		"SELECT username, planet_name FROM users WHERE username = $1",
		username,
	).Scan(&retrievedUsername, &retrievedPlanet)

	if err != nil {
		log.Fatal("Failed to retrieve user:", err)
	}

	fmt.Println("Retrieved user:", retrievedUsername)
	fmt.Println("Planet:", retrievedPlanet)
	var userID int64

	err = pool.QueryRow(ctx,
		"SELECT id FROM users WHERE username = $1",
		username,
	).Scan(&userID)

	if err != nil {
		log.Fatal("Failed to retrieve user ID:", err)
	}

	postTitle := "Hello, Planet Earth!"
	postContent := "My first post on Planetary."

	latitude := 40.7128
	longitude := -74.0060

	_, err = pool.Exec(ctx,
		`INSERT INTO posts
    (user_id, title, content, latitude, longitude)
    VALUES ($1, $2, $3, $4, $5)`,
		userID, postTitle, postContent, latitude, longitude,
	)

	if err != nil {
		log.Fatal("Failed to create post:", err)
	}

	fmt.Println("Successfully created post:", postTitle)
	rows, err := pool.Query(ctx,
		`SELECT users.username, posts.title, posts.content,
            posts.latitude, posts.longitude
     FROM posts
     JOIN users ON posts.user_id = users.id
     ORDER BY posts.created_at DESC`,
	)

	if err != nil {
		log.Fatal("Failed to retrieve posts:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var author, title, content string
		var lat, lon float64

		err := rows.Scan(&author, &title, &content, &lat, &lon)
		if err != nil {
			log.Fatal("Failed to read post:", err)
		}

		fmt.Printf("\nAuthor: %s\nTitle: %s\nContent: %s\nLocation: %f, %f\n",
			author, title, content, lat, lon)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error iterating through posts:", err)
	}
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Planetary backend is running!")

	})
	fmt.Println("Server starting on port 8080...")

	log.Fatal(http.ListenAndServe(":8080", nil))
	//fmt.Println("Welcome to the Planetary Game!")
}
