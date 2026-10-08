package storage

import (
	"testing"
)

func TestURLStore_SaveAndGet(t *testing.T) {
	store := NewURLStore()

	// Test Save
	code, err := store.Save("https://example.com", "code123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "code123" {
		t.Errorf("expected code123, got %s", code)
	}

	// Test Save duplicate URL
	code2, err := store.Save("https://example.com", "code456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code2 != "code123" {
		t.Errorf("expected code123 for existing URL, got %s", code2)
	}

	// Test Save collision code
	_, err = store.Save("https://example.org", "code123")
	if err != ErrCodeCollision {
		t.Errorf("expected ErrCodeCollision, got %v", err)
	}

	// Test GetByCode
	data, err := store.GetByCode("code123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data.URL != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", data.URL)
	}

	// Test GetByCode NotFound
	_, err = store.GetByCode("unknown")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// Test GetByURL
	fetchedCode, err := store.GetByURL("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fetchedCode != "code123" {
		t.Errorf("expected code123, got %s", fetchedCode)
	}

	// Test GetByURL NotFound
	_, err = store.GetByURL("https://unknown.com")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
