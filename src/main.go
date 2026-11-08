package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Post struct {
	ID        int64   `json:"id"`
	Author    string  `json:"author"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Online    bool    `json:"online"`
}

type CreatePostRequest struct {
	Username  string  `json:"username"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Comment struct {
	ID              int64  `json:"id"`
	PostID          int64  `json:"post_id"`
	Author          string `json:"author"`
	Content         string `json:"content"`
	ParentCommentID *int64 `json:"parent_comment_id"`
}

type CreateCommentRequest struct {
	Username        string `json:"username"`
	Content         string `json:"content"`
	ParentCommentID *int64 `json:"parent_comment_id"`
}

type UserProfile struct {
	Username          string `json:"username"`
	PlanetName        string `json:"planet_name"`
	Bio               string `json:"bio"`
	PlanetDescription string `json:"planet_description"`
}

type UpdateProfileRequest struct {
	Username          string `json:"username"`
	Bio               string `json:"bio"`
	PlanetName        string `json:"planet_name"`
	PlanetDescription string `json:"planet_description"`
}

type ChatChannel struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ChatMessage struct {
	ID        int64     `json:"id"`
	ChannelID int64     `json:"channel_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Online    bool      `json:"online"`
}

type CreateChatMessageRequest struct {
	Username string `json:"username"`
	Content  string `json:"content"`
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

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("Could not connect to Supabase:", err)
	}

	fmt.Println("Successfully connected to Supabase!")

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

	fmt.Println("Successfully initialized user:", username)

	// User profiles
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		queryCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var profile UserProfile

		if r.Method == http.MethodPut {
			var request UpdateProfileRequest

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if request.Username == "" || request.PlanetName == "" {
				http.Error(w, "Username and planet name are required", http.StatusBadRequest)
				return
			}

			err := pool.QueryRow(queryCtx,
				`UPDATE users
				SET bio = $1, planet_name = $2, planet_description = $3
				WHERE username = $4
				RETURNING username, planet_name, bio, planet_description`,
				request.Bio,
				request.PlanetName,
				request.PlanetDescription,
				request.Username,
			).Scan(
				&profile.Username,
				&profile.PlanetName,
				&profile.Bio,
				&profile.PlanetDescription,
			)

			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, "User not found", http.StatusNotFound)
				} else {
					log.Println("Failed to update profile:", err)
					http.Error(w, "Failed to update profile", http.StatusInternalServerError)
				}
				return
			}
		} else {
			requestedUsername := r.URL.Query().Get("username")

			if requestedUsername == "" {
				http.Error(w, "Missing username", http.StatusBadRequest)
				return
			}

			err := pool.QueryRow(queryCtx,
				`SELECT username, planet_name, bio, planet_description
				FROM users
				WHERE username = $1`,
				requestedUsername,
			).Scan(
				&profile.Username,
				&profile.PlanetName,
				&profile.Bio,
				&profile.PlanetDescription,
			)

			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, "User not found", http.StatusNotFound)
				} else {
					log.Println("Failed to retrieve user profile:", err)
					http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
				}
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	})

	// User heartbeat
	http.HandleFunc("/users/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		requestedUsername := r.URL.Query().Get("username")
		if requestedUsername == "" {
			http.Error(w, "Username is required", http.StatusBadRequest)
			return
		}

		queryCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		result, err := pool.Exec(queryCtx,
			"UPDATE users SET last_seen_at = NOW() WHERE username = $1",
			requestedUsername,
		)

		if err != nil {
			log.Println("Failed to update online status:", err)
			http.Error(w, "Failed to update online status", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected() == 0 {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Posts
	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet &&
			r.Method != http.MethodPost &&
			r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		queryCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if r.Method == http.MethodDelete {
			postID, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || postID <= 0 {
				http.Error(w, "Invalid or missing post id", http.StatusBadRequest)
				return
			}

			result, err := pool.Exec(queryCtx,
				`DELETE FROM posts
				USING users
				WHERE posts.user_id = users.id
				AND posts.id = $1
				AND users.username = $2`,
				postID,
				username,
			)

			if err != nil {
				log.Println("Failed to delete post:", err)
				http.Error(w, "Failed to delete post", http.StatusInternalServerError)
				return
			}

			if result.RowsAffected() == 0 {
				http.Error(w, "Post not found or cannot be deleted", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Post deleted successfully",
				"post_id": postID,
			})
			return
		}

		if r.Method == http.MethodPost {
			var request CreatePostRequest

			err := json.NewDecoder(r.Body).Decode(&request)
			if err != nil {
				log.Println("JSON decoding error:", err)
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if request.Username == "" || request.Title == "" || request.Content == "" {
				http.Error(w, "Missing required fields", http.StatusBadRequest)
				return
			}

			if request.Latitude < -90 || request.Latitude > 90 ||
				request.Longitude < -180 || request.Longitude > 180 {
				http.Error(w, "Invalid coordinates", http.StatusBadRequest)
				return
			}

			var userID int64

			err = pool.QueryRow(queryCtx,
				"SELECT id FROM users WHERE username = $1",
				request.Username,
			).Scan(&userID)

			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, "User not found", http.StatusNotFound)
				} else {
					log.Println("Failed to retrieve user:", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
				}
				return
			}

			var postID int64

			err = pool.QueryRow(queryCtx,
				`INSERT INTO posts
				(user_id, title, content, latitude, longitude)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`,
				userID,
				request.Title,
				request.Content,
				request.Latitude,
				request.Longitude,
			).Scan(&postID)

			if err != nil {
				log.Println("Failed to create post:", err)
				http.Error(w, "Failed to create post", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Post created successfully",
				"post_id": postID,
			})
			return
		}

		rows, err := pool.Query(queryCtx,
			`SELECT posts.id, users.username, posts.title, posts.content,
			posts.latitude, posts.longitude,
			COALESCE(users.last_seen_at > NOW() - INTERVAL '60 seconds', FALSE)
			FROM posts
			JOIN users ON posts.user_id = users.id
			ORDER BY posts.created_at DESC`,
		)

		if err != nil {
			log.Println("Failed to retrieve posts:", err)
			http.Error(w, "Failed to retrieve posts", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		posts := make([]Post, 0)

		for rows.Next() {
			var post Post

			err := rows.Scan(
				&post.ID,
				&post.Author,
				&post.Title,
				&post.Content,
				&post.Latitude,
				&post.Longitude,
				&post.Online,
			)

			if err != nil {
				log.Println("Failed to read posts:", err)
				http.Error(w, "Failed to read posts", http.StatusInternalServerError)
				return
			}

			posts = append(posts, post)
		}

		if err := rows.Err(); err != nil {
			log.Println("Error iterating through posts:", err)
			http.Error(w, "Error reading posts", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(posts)
	})

	// Comments
	http.HandleFunc("/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		postID, err := strconv.ParseInt(r.URL.Query().Get("post_id"), 10, 64)
		if err != nil || postID <= 0 {
			http.Error(w, "Invalid or missing post_id", http.StatusBadRequest)
			return
		}

		queryCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if r.Method == http.MethodPost {
			var request CreateCommentRequest

			err := json.NewDecoder(r.Body).Decode(&request)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if request.Username == "" || request.Content == "" {
				http.Error(w, "Missing required fields", http.StatusBadRequest)
				return
			}

			var userID int64

			err = pool.QueryRow(queryCtx,
				"SELECT id FROM users WHERE username = $1",
				request.Username,
			).Scan(&userID)

			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, "User not found", http.StatusNotFound)
				} else {
					log.Println("Failed to retrieve user:", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
				}
				return
			}

			var postExists bool

			err = pool.QueryRow(queryCtx,
				"SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1)",
				postID,
			).Scan(&postExists)

			if err != nil {
				log.Println("Failed to check post:", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			if !postExists {
				http.Error(w, "Post not found", http.StatusNotFound)
				return
			}

			if request.ParentCommentID != nil {
				var parentExists bool

				err = pool.QueryRow(queryCtx,
					`SELECT EXISTS(
						SELECT 1 FROM comments
						WHERE id = $1 AND post_id = $2
					)`,
					*request.ParentCommentID,
					postID,
				).Scan(&parentExists)

				if err != nil {
					log.Println("Failed to check parent comment:", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
					return
				}

				if !parentExists {
					http.Error(w, "Parent comment not found for this post", http.StatusNotFound)
					return
				}
			}

			var commentID int64

			err = pool.QueryRow(queryCtx,
				`INSERT INTO comments
				(post_id, user_id, parent_comment_id, content)
				VALUES ($1, $2, $3, $4)
				RETURNING id`,
				postID,
				userID,
				request.ParentCommentID,
				request.Content,
			).Scan(&commentID)

			if err != nil {
				log.Println("Failed to create comment:", err)
				http.Error(w, "Failed to create comment", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"message":    "Comment created successfully",
				"comment_id": commentID,
			})
			return
		}

		rows, err := pool.Query(queryCtx,
			`SELECT comments.id, comments.post_id, users.username,
			comments.content, comments.parent_comment_id
			FROM comments
			JOIN users ON comments.user_id = users.id
			WHERE comments.post_id = $1
			ORDER BY comments.created_at ASC`,
			postID,
		)

		if err != nil {
			log.Println("Failed to retrieve comments:", err)
			http.Error(w, "Failed to retrieve comments", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		comments := make([]Comment, 0)

		for rows.Next() {
			var comment Comment

			err := rows.Scan(
				&comment.ID,
				&comment.PostID,
				&comment.Author,
				&comment.Content,
				&comment.ParentCommentID,
			)

			if err != nil {
				log.Println("Failed to read comments:", err)
				http.Error(w, "Failed to read comments", http.StatusInternalServerError)
				return
			}

			comments = append(comments, comment)
		}

		if err := rows.Err(); err != nil {
			log.Println("Error iterating through comments:", err)
			http.Error(w, "Error reading comments", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(comments)
	})

	// Chat channels
	http.HandleFunc("/chat/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		queryCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rows, err := pool.Query(queryCtx,
			`SELECT id, name, description
			FROM chat_channels
			ORDER BY id ASC`,
		)

		if err != nil {
			log.Println("Failed to retrieve chat channels:", err)
			http.Error(w, "Failed to retrieve chat channels", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		channels := make([]ChatChannel, 0)

		for rows.Next() {
			var channel ChatChannel

			if err := rows.Scan(
				&channel.ID,
				&channel.Name,
				&channel.Description,
			); err != nil {
				log.Println("Failed to read chat channel:", err)
				http.Error(w, "Failed to read chat channels", http.StatusInternalServerError)
				return
			}

			channels = append(channels, channel)
		}

		if err := rows.Err(); err != nil {
			log.Println("Error iterating through chat channels:", err)
			http.Error(w, "Error reading chat channels", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(channels)
	})

	// Chat messages
	http.HandleFunc("/chat/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		channelID, err := strconv.ParseInt(r.URL.Query().Get("channel_id"), 10, 64)
		if err != nil || channelID <= 0 {
			http.Error(w, "Invalid or missing channel_id", http.StatusBadRequest)
			return
		}

		queryCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		if r.Method == http.MethodPost {
			var request CreateChatMessageRequest

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			request.Content = strings.TrimSpace(request.Content)

			if request.Username == "" || request.Content == "" {
				http.Error(w, "Username and message content are required", http.StatusBadRequest)
				return
			}

			if len([]rune(request.Content)) > 1000 {
				http.Error(w, "Message cannot exceed 1000 characters", http.StatusBadRequest)
				return
			}

			var userID int64

			err := pool.QueryRow(queryCtx,
				"SELECT id FROM users WHERE username = $1",
				request.Username,
			).Scan(&userID)

			if err != nil {
				if err == pgx.ErrNoRows {
					http.Error(w, "User not found", http.StatusNotFound)
				} else {
					log.Println("Failed to retrieve chat user:", err)
					http.Error(w, "Database error", http.StatusInternalServerError)
				}
				return
			}

			var messageID int64

			err = pool.QueryRow(queryCtx,
				`INSERT INTO chat_messages (channel_id, user_id, content)
				VALUES ($1, $2, $3)
				RETURNING id`,
				channelID,
				userID,
				request.Content,
			).Scan(&messageID)

			if err != nil {
				log.Println("Failed to create chat message:", err)
				http.Error(w, "Failed to create message. Check that the channel exists.", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"message":    "Chat message sent successfully",
				"message_id": messageID,
			})
			return
		}

		rows, err := pool.Query(queryCtx,
			`SELECT id, channel_id, username, content, created_at, online
			FROM (
				SELECT chat_messages.id, chat_messages.channel_id,
					users.username, chat_messages.content,
					chat_messages.created_at,
					COALESCE(users.last_seen_at > NOW() - INTERVAL '60 seconds', FALSE) AS online
				FROM chat_messages
				JOIN users ON chat_messages.user_id = users.id
				WHERE chat_messages.channel_id = $1
				ORDER BY chat_messages.created_at DESC, chat_messages.id DESC
				LIMIT 100
			) recent_messages
			ORDER BY created_at ASC, id ASC`,
			channelID,
		)

		if err != nil {
			log.Println("Failed to retrieve chat messages:", err)
			http.Error(w, "Failed to retrieve chat messages", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		messages := make([]ChatMessage, 0)

		for rows.Next() {
			var message ChatMessage

			if err := rows.Scan(
				&message.ID,
				&message.ChannelID,
				&message.Author,
				&message.Content,
				&message.CreatedAt,
				&message.Online,
			); err != nil {
				log.Println("Failed to read chat message:", err)
				http.Error(w, "Failed to read chat messages", http.StatusInternalServerError)
				return
			}

			messages = append(messages, message)
		}

		if err := rows.Err(); err != nil {
			log.Println("Error iterating through chat messages:", err)
			http.Error(w, "Error reading chat messages", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	})

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Planetary backend is running!")
	})

	fmt.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
