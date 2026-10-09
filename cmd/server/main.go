package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ParhamAmiri06/url-shortener/internal/api"
	"github.com/ParhamAmiri06/url-shortener/internal/storage"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it")
	}

	var store api.Store

	if getEnvOrDefault("USE_DB", "true") == "true" {
		dbDSN := os.Getenv("DB_DSN")
		if dbDSN == "" {
			host := getEnvOrDefault("DB_HOST", "localhost")
			user := getEnvOrDefault("DB_USER", "postgres")
			password := getEnvOrDefault("DB_PASSWORD", "postgres")
			dbname := getEnvOrDefault("DB_NAME", "shortener")
			port := getEnvOrDefault("DB_PORT", "5432")
			sslmode := getEnvOrDefault("DB_SSLMODE", "disable")
			fmt.Println("tehran")
			dbDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Tehran",
				host, user, password, dbname, port, sslmode)
		}

		fmt.Println("Final dbDSN:", dbDSN)
		db, err := gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
		if err != nil {
			log.Fatalf("failed to connect database: %v", err)
		}
		dbStore, err := storage.NewDbStore(db)
		if err != nil {
			log.Fatalf("failed to initialize db store: %v", err)
		}
		store = dbStore
		fmt.Println("Using db store")

	} else {
		fmt.Println("Using in-memory store")
		store = storage.NewURLStore()
	}

	baseURL := getEnvOrDefault("BASE_URL", "http://localhost:8080")

	handler := &api.Handler{
		Store:   store,
		BaseURL: baseURL,
	}

	rateLimiter := api.NewIPRateLimiter(rate.Limit(1), 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/shorten", api.RateLimitMiddleware(rateLimiter, handler.Shorten))
	mux.HandleFunc("/api/v1/links/", handler.GetLinkStats)
	mux.HandleFunc("/", handler.Redirect)

	port := getEnvOrDefault("SERVER_PORT", "8080")
	addr := ":" + port

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", addr)
	log.Printf("Base URL is %s", baseURL)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting gracefully")
}
