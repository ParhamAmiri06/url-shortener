package storage

import (
	"errors"
	"sync"
	"time"
)

var ErrCodeCollision = errors.New("code collision")
var ErrNotFound = errors.New("not found")

type LinkData struct {
	URL       string
	CreatedAt time.Time
}

type URLStore struct {
	mu        sync.RWMutex
	codeToURL map[string]LinkData
	urlToCode map[string]string
}

func NewURLStore() *URLStore {
	return &URLStore{
		codeToURL: make(map[string]LinkData),
		urlToCode: make(map[string]string),
	}
}

func (s *URLStore) GetByCode(code string) (LinkData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.codeToURL[code]
	if !ok {
		return LinkData{}, ErrNotFound
	}
	return data, nil
}

func (s *URLStore) GetByURL(url string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.urlToCode[url]
	if !ok {
		return "", ErrNotFound
	}
	return code, nil
}

func (s *URLStore) Save(url, code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingCode, ok := s.urlToCode[url]; ok {
		return existingCode, nil
	}

	if _, ok := s.codeToURL[code]; ok {
		return "", ErrCodeCollision
	}

	s.codeToURL[code] = LinkData{
		URL:       url,
		CreatedAt: time.Now().UTC(),
	}
	s.urlToCode[url] = code

	return code, nil
}
