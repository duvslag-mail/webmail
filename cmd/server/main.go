package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/duvlsag-mail/webmail/db/sqlc"
	"github.com/duvlsag-mail/webmail/internal/auth"
	"github.com/duvlsag-mail/webmail/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {
	// load .env config
	godotenv.Load()

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	imapEncryptionKey := os.Getenv("IMAP_ENCRYPTION_KEY")

	//  db setup
	dbUrl := "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// ping db
	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// migrations

	stdDB, err := sql.Open("pgx", dbUrl)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose: failed to set dialect: %v", err)
	}
	if err := goose.Up(stdDB, "db/migrations"); err != nil {
		log.Fatalf("goose: failed to run migrations: %v", err)
	} else {
		log.Println("goose: migrations applied successfully")
	}
	defer stdDB.Close()

	queries := sqlc.New(pool)

	// http routing
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	authService := auth.NewAuthService(queries, []byte(imapEncryptionKey))
	authHandler := handlers.NewAuthHandler(authService)
	authHandler.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("-----------")
	log.Printf("starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
