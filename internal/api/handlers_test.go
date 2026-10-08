package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/ParhamAmiri06/url-shortener/internal/storage"
)

type fakeStore struct {
	mu        sync.RWMutex
	codeToURL map[string]storage.LinkData
	urlToCode map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		codeToURL: make(map[string]storage.LinkData),
		urlToCode: make(map[string]string),
	}
}

func (s *fakeStore) GetByCode(code string) (storage.LinkData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if code == "trigger-error" {
		return storage.LinkData{}, errors.New("simulated error")
	}
	data, ok := s.codeToURL[code]
	if !ok {
		return storage.LinkData{}, storage.ErrNotFound
	}
	return data, nil
}

func (s *fakeStore) GetByURL(url string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.urlToCode[url]
	if !ok {
		return "", storage.ErrNotFound
	}
	return code, nil
}

func (s *fakeStore) Save(url, code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existingCode, ok := s.urlToCode[url]; ok {
		return existingCode, nil
	}
	if _, ok := s.codeToURL[code]; ok {
		return "", storage.ErrCodeCollision
	}
	s.codeToURL[code] = storage.LinkData{URL: url}
	s.urlToCode[url] = code
	return code, nil
}

func TestShortenAndRedirect(t *testing.T) {
	store := newFakeStore()
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
	store := newFakeStore()
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
	store := newFakeStore()
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
	store := newFakeStore()
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

func TestGetLinkStats(t *testing.T) {
	store := newFakeStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	// 1. Shorten a URL to get a code
	reqBody, _ := json.Marshal(ShortenRequest{URL: "https://example.com/stats"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()
	handler.Shorten(rec, req)

	var resp ShortenResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	// 2. Test GetLinkStats for the existing code
	reqStats := httptest.NewRequest(http.MethodGet, "/api/v1/links/"+resp.Code, nil)
	recStats := httptest.NewRecorder()
	handler.GetLinkStats(recStats, reqStats)

	if recStats.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, recStats.Code)
	}

	var stats map[string]string
	json.NewDecoder(recStats.Body).Decode(&stats)
	if stats["url"] != "https://example.com/stats" {
		t.Errorf("expected url https://example.com/stats, got %s", stats["url"])
	}
	if stats["created_at"] == "" {
		t.Errorf("expected non-empty created_at")
	}

	// 3. Test GetLinkStats for unknown code
	reqUnknown := httptest.NewRequest(http.MethodGet, "/api/v1/links/unknown-code", nil)
	recUnknown := httptest.NewRecorder()
	handler.GetLinkStats(recUnknown, reqUnknown)

	if recUnknown.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, recUnknown.Code)
	}
}

func TestRedirectEdgeCases(t *testing.T) {
	store := newFakeStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	tests := []struct {
		name       string
		method     string
		url        string
		expectCode int
	}{
		{"Method Not Allowed", http.MethodPost, "/some-code", http.StatusMethodNotAllowed},
		{"Empty Code", http.MethodGet, "/", http.StatusNotFound},
		{"Internal Server Error", http.MethodGet, "/trigger-error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			rec := httptest.NewRecorder()
			handler.Redirect(rec, req)

			if rec.Code != tt.expectCode {
				t.Errorf("expected status %d, got %d", tt.expectCode, rec.Code)
			}
		})
	}
}

func TestShortenEdgeCases(t *testing.T) {
	store := newFakeStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	t.Run("Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/shorten", nil)
		rec := httptest.NewRecorder()
		handler.Shorten(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString("{invalid-json}"))
		rec := httptest.NewRecorder()
		handler.Shorten(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}

func TestGetLinkStatsEdgeCases(t *testing.T) {
	store := newFakeStore()
	handler := &Handler{
		Store:   store,
		BaseURL: "http://localhost:8080",
	}

	t.Run("Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/links/code", nil)
		rec := httptest.NewRecorder()
		handler.GetLinkStats(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/trigger-error", nil)
		rec := httptest.NewRecorder()
		handler.GetLinkStats(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
		}
	})

	t.Run("Empty Code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/links/", nil)
		rec := httptest.NewRecorder()
		handler.GetLinkStats(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}
