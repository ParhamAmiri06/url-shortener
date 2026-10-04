package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ParhamAmiri06/url-shortener/internal/shortener"
	"github.com/ParhamAmiri06/url-shortener/internal/storage"
)

type Store interface {
	GetByCode(code string) (storage.LinkData, error)
	GetByURL(url string) (string, error)
	Save(url, code string) (string, error)
}

type Handler struct {
	Store   Store
	BaseURL string
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
}

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	normalized, err := shortener.NormalizeURL(req.URL)
	if err != nil {
		if errors.Is(err, shortener.ErrInvalidURL) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if existingCode, err := h.Store.GetByURL(normalized); err == nil {
		resp := ShortenResponse{
			Code:     existingCode,
			ShortURL: fmt.Sprintf("%s/%s", strings.TrimRight(h.BaseURL, "/"), existingCode),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
		return
	}

	var code string
	for {
		newCode, err := shortener.GenerateCode()
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		savedCode, err := h.Store.Save(normalized, newCode)
		if err == storage.ErrCodeCollision {
			continue
		}

		code = savedCode
		break
	}

	resp := ShortenResponse{
		Code:     code,
		ShortURL: fmt.Sprintf("%s/%s", strings.TrimRight(h.BaseURL, "/"), code),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	data, err := h.Store.GetByCode(code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(w, r, data.URL, http.StatusFound)
}

func (h *Handler) GetLinkStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := strings.TrimPrefix(r.URL.Path, "/api/v1/links/")
	if code == "" || code == r.URL.Path {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Unknown code"))
		return
	}

	data, err := h.Store.GetByCode(code)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Unknown code"))
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	type StatsResponse struct {
		URL       string `json:"url"`
		CreatedAt string `json:"created_at"`
	}

	resp := StatsResponse{
		URL:       data.URL,
		CreatedAt: data.CreatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
