package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkShorten(b *testing.B) {
	store := newFakeStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	reqBody, _ := json.Marshal(ShortenRequest{URL: "https://example.com/bench"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()
		handler.Shorten(rec, req)
	}
}

func BenchmarkRedirect(b *testing.B) {
	store := newFakeStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	reqBody, _ := json.Marshal(ShortenRequest{URL: "https://example.com/bench-redirect"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	handler.Shorten(rec, req)

	var resp ShortenResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	code := resp.Code

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+code, nil)
		rec := httptest.NewRecorder()
		handler.Redirect(rec, req)
	}
}
