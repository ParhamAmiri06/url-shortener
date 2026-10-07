package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ParhamAmiri06/url-shortener/internal/api"
	"github.com/ParhamAmiri06/url-shortener/internal/storage"
	"github.com/joho/godotenv"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/shorten", handler.Shorten)
	mux.HandleFunc("/api/v1/links/", handler.GetLinkStats)
	mux.HandleFunc("/", handler.Redirect)

	port := getEnvOrDefault("SERVER_PORT", "8080")
	addr := ":" + port

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", addr)
	log.Printf("Base URL is %s", baseURL)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
