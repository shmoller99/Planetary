package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// db is the shared connection pool, set in main after connecting.
var db *pgxpool.Pool

const maxNameLength = 16

type Post struct {
	ID        int64
	UserID    int64
	Title     string
	Message   string
	Longitude float64
	Latitude  float64
	CreatedAt time.Time
}

type User struct {
	ID         int64
	Username   string
	PlanetName string
}

// MakePost inserts a new post for the given user. The database assigns the
// unique post ID, which is returned on the Post.
func MakePost(ctx context.Context, userID int64, title, message string, longitude, latitude float64) (*Post, error) {
	p := &Post{
		UserID:    userID,
		Title:     title,
		Message:   message,
		Longitude: longitude,
		Latitude:  latitude,
	}
	err := db.QueryRow(ctx,
		`INSERT INTO posts (user_id, title, content, latitude, longitude)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		userID, title, message, latitude, longitude,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("make post: %w", err)
	}
	return p, nil
}

// ReadPost looks up a post by ID. It returns nil, nil if no such post exists.
func ReadPost(ctx context.Context, postID int64) (*Post, error) {
	p := &Post{}
	err := db.QueryRow(ctx,
		`SELECT id, user_id, title, content, longitude, latitude, created_at
		 FROM posts WHERE id = $1`,
		postID,
	).Scan(&p.ID, &p.UserID, &p.Title, &p.Message, &p.Longitude, &p.Latitude, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read post %d: %w", postID, err)
	}
	return p, nil
}

// NewUser creates a user with a unique username and unique planet name, each
// at most 16 characters. The database assigns the unique user ID.
func NewUser(ctx context.Context, username, planetName string) (*User, error) {
	if username == "" || utf8.RuneCountInString(username) > maxNameLength {
		return nil, fmt.Errorf("username must be 1-%d characters", maxNameLength)
	}
	if planetName == "" || utf8.RuneCountInString(planetName) > maxNameLength {
		return nil, fmt.Errorf("planet name must be 1-%d characters", maxNameLength)
	}

	var taken string
	err := db.QueryRow(ctx,
		`SELECT CASE WHEN username = $1 THEN 'username' ELSE 'planet name' END
		 FROM users WHERE username = $1 OR planet_name = $2 LIMIT 1`,
		username, planetName,
	).Scan(&taken)
	if err == nil {
		return nil, fmt.Errorf("%s is already taken", taken)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("new user: %w", err)
	}

	u := &User{Username: username, PlanetName: planetName}
	err = db.QueryRow(ctx,
		`INSERT INTO users (username, planet_name) VALUES ($1, $2) RETURNING id`,
		username, planetName,
	).Scan(&u.ID)
	if err != nil {
		// Covers the race where another request claims a name between the check and the insert.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, errors.New("username or planet name is already taken")
		}
		return nil, fmt.Errorf("new user: %w", err)
	}
	return u, nil
}

// FindUser looks up a user by ID. It returns nil, nil if no such user exists.
func FindUser(ctx context.Context, userID int64) (*User, error) {
	u := &User{}
	err := db.QueryRow(ctx,
		`SELECT id, username, planet_name FROM users WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Username, &u.PlanetName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user %d: %w", userID, err)
	}
	return u, nil
}

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
	db = pool
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
