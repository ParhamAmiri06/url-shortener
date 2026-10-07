package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDbStore_RestartSimulation(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "url-shortener-db-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")

	t.Run("Instance 1: Save data", func(t *testing.T) {
		db1, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open db1: %v", err)
		}

		store1, err := NewDbStore(db1)
		if err != nil {
			t.Fatalf("Failed to create store1: %v", err)
		}

		code, err := store1.Save("https://example.com/persist", "pers12")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if code != "pers12" {
			t.Errorf("Expected code pers12, got %s", code)
		}

		sqlDB, err := db1.DB()
		if err == nil {
			sqlDB.Close()
		}
	})

	t.Run("Instance 2: Read data after restart", func(t *testing.T) {
		db2, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("Failed to open db2: %v", err)
		}

		store2, err := NewDbStore(db2)
		if err != nil {
			t.Fatalf("Failed to create store2: %v", err)
		}

		linkData, err := store2.GetByCode("pers12")
		if err != nil {
			t.Fatalf("Failed to get link by code after restart: %v", err)
		}

		if linkData.URL != "https://example.com/persist" {
			t.Errorf("Expected URL %s, got %s", "https://example.com/persist", linkData.URL)
		}

		code, err := store2.GetByURL("https://example.com/persist")
		if err != nil {
			t.Fatalf("Failed to get code by URL after restart: %v", err)
		}
		if code != "pers12" {
			t.Errorf("Expected code %s, got %s", "pers12", code)
		}
	})
}

func TestDbStore_Concurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "url-shortener-db-test-conc")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_conc.db")
	// Enable WAL mode to allow concurrent readers/writers in SQLite
	db, err := gorm.Open(sqlite.Open(dbPath+"?_journal_mode=WAL"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}

	store, err := NewDbStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	const numRoutines = 5
	var wg sync.WaitGroup
	wg.Add(numRoutines)

	for i := 0; i < numRoutines; i++ {
		go func(i int) {
			defer wg.Done()
			url := fmt.Sprintf("https://example.com/conc/%d", i)
			code := fmt.Sprintf("c%d", i)

			_, err := store.Save(url, code)
			if err != nil {
				t.Errorf("Concurrent Save failed: %v", err)
			}

			_, err = store.GetByCode(code)
			if err != nil {
				t.Errorf("Concurrent GetByCode failed: %v", err)
			}
		}(i)
	}
	wg.Wait()
}
