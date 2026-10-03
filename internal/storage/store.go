package storage

import "sync"

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

func (s *URLStore) Save(url, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codeToURL[code] = url
	s.urlToCode[url] = code
}
