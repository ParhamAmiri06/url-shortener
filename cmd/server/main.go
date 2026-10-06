package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/ParhamAmiri06/url-shortener/internal/api"
	"github.com/ParhamAmiri06/url-shortener/internal/storage"
)

var (
	addrFlag = flag.String("addr", ":8080", "address to listen on")
	baseFlag = flag.String("base", "http://localhost:8080", "base URL for short links")
)

func main() {
	flag.Parse()

	store := storage.NewURLStore()
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
