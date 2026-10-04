package shortener

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var ErrInvalidURL = errors.New("invalid url")

func GenerateCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = alphabet[b[i]%byte(len(alphabet))]
	}
	return string(b), nil
}

func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("url cannot be empty: %w", ErrInvalidURL)
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", fmt.Errorf("invalid url format: %w", ErrInvalidURL)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("invalid scheme: must be http or https: %w", ErrInvalidURL)
	}

	hostname := strings.ToLower(u.Hostname())
	port := u.Port()

	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}

	if port != "" {
		u.Host = hostname + ":" + port
	} else {
		u.Host = hostname
	}

	if u.Path == "/" {
		u.Path = ""
	}

	return u.String(), nil
}
