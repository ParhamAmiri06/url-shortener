package storage

import (
	"errors"
	"sync"
)

var ErrCodeCollision = errors.New("code collision")

type URLStore struct {
	mu        sync.RWMutex
	codeToURL map[string]string
	urlToCode map[string]string
}

func NewURLStore() *URLStore {
	return &URLStore{
		codeToURL: make(map[string]string),
		urlToCode: make(map[string]string),
	}
}

func (s *URLStore) GetByCode(code string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.codeToURL[code]
	return url, ok
}

func (s *URLStore) GetByURL(url string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.urlToCode[url]
	return code, ok
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
	
	s.codeToURL[code] = url
	s.urlToCode[url] = code
	
	return code, nil
}
