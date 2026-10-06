package main

import (
	"flag"
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

var (
	addrFlag = flag.String("addr", ":8080", "address to listen on")
	baseFlag = flag.String("base", "http://localhost:8080", "base URL for short links")
)

func getEnvOrDefault(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	flag.Parse()

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

			dbDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
				host, user, password, dbname, port, sslmode)
		}

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

	handler := &api.Handler{
		Store:   store,
		BaseURL: *baseFlag,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/shorten", handler.Shorten)
	mux.HandleFunc("/api/v1/links/", handler.GetLinkStats)
	mux.HandleFunc("/", handler.Redirect)

	srv := &http.Server{
		Addr:         *addrFlag,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 3 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on %s", *addrFlag)
	log.Printf("Base URL is %s", *baseFlag)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
