package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ParhamAmiri06/url-shortener/internal/storage"
)

func TestShortenAndRedirect(t *testing.T) {
	store := storage.NewURLStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	reqBody, _ := json.Marshal(ShortenRequest{URL: "https://example.com/test"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	handler.Shorten(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var resp ShortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Code == "" || resp.ShortURL == "" {
		t.Errorf("expected non-empty code and short_url, got %+v", resp)
	}

	// Test Redirect
	req = httptest.NewRequest(http.MethodGet, "/"+resp.Code, nil)
	rec = httptest.NewRecorder()
	handler.Redirect(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d for redirect, got %d", http.StatusFound, rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "https://example.com/test" {
		t.Errorf("expected location https://example.com/test, got %s", loc)
	}
}

func TestIdempotency(t *testing.T) {
	store := storage.NewURLStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	reqBody, _ := json.Marshal(ShortenRequest{URL: "https://example.com/idemp"})

	// First POST
	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	rec1 := httptest.NewRecorder()
	handler.Shorten(rec1, req1)
	var resp1 ShortenResponse
	json.NewDecoder(rec1.Body).Decode(&resp1)

	// Second POST
	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	rec2 := httptest.NewRecorder()
	handler.Shorten(rec2, req2)
	var resp2 ShortenResponse
	json.NewDecoder(rec2.Body).Decode(&resp2)

	if resp1.Code != resp2.Code {
		t.Errorf("expected idempotency, got different codes: %s vs %s", resp1.Code, resp2.Code)
	}
}

func TestTableBadURLAndUnknownCode(t *testing.T) {
	store := storage.NewURLStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	tests := []struct {
		name       string
		method     string
		url        string
		body       interface{}
		expectCode int
	}{
		{"Bad URL Format", http.MethodPost, "/api/shorten", ShortenRequest{URL: "not-a-url"}, http.StatusBadRequest},
		{"Missing Scheme", http.MethodPost, "/api/shorten", ShortenRequest{URL: "example.com"}, http.StatusBadRequest},
		{"Empty URL", http.MethodPost, "/api/shorten", ShortenRequest{URL: ""}, http.StatusBadRequest},
		{"Unknown Code", http.MethodGet, "/unknown123", nil, http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				b, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer(b))
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			rec := httptest.NewRecorder()
			if tt.method == http.MethodPost {
				handler.Shorten(rec, req)
			} else {
				handler.Redirect(rec, req)
			}

			if rec.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, rec.Code)
			}
		})
	}
}

func TestConcurrentDuplicateShorten(t *testing.T) {
	store := storage.NewURLStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	const concurrency = 100
	url := "https://example.com/concurrent"
	reqBody, _ := json.Marshal(ShortenRequest{URL: url})

	codes := make([]string, concurrency)
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
			rec := httptest.NewRecorder()
			handler.Shorten(rec, req)

			var resp ShortenResponse
			json.NewDecoder(rec.Body).Decode(&resp)
			codes[idx] = resp.Code
		}(i)
	}
	wg.Wait()

	firstCode := codes[0]
	if firstCode == "" {
		t.Fatalf("expected non-empty code")
	}
	for i := 1; i < concurrency; i++ {
		if codes[i] != firstCode {
			t.Errorf("expected all concurrent requests to return same code %s, but got %s at index %d", firstCode, codes[i], i)
		}
	}
}
