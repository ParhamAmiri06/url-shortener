package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ParhamAmiri06/url-shortener/internal/shortener"
	"github.com/ParhamAmiri06/url-shortener/internal/storage"
)

type Handler struct {
	Store   *storage.URLStore
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if existingCode, ok := h.Store.GetByURL(normalized); ok {
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

	longURL, ok := h.Store.GetByCode(code)
	if !ok {
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}
