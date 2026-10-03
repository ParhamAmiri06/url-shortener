package main

import (
	"flag"
	"log"
	"net/http"

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

	http.HandleFunc("/api/shorten", handler.Shorten)
	http.HandleFunc("/", handler.Redirect)

	log.Printf("Starting server on %s", *addrFlag)
	log.Printf("Base URL is %s", *baseFlag)
	if err := http.ListenAndServe(*addrFlag, nil); err != nil {
		log.Fatal(err)
	}
}
