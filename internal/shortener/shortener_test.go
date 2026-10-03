package shortener

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		expectURL string
		expectErr bool
	}{
		{"Valid HTTP", "http://example.com/path", "http://example.com/path", false},
		{"Valid HTTPS", "https://example.com/path", "https://example.com/path", false},
		{"Lowercase Domain", "HTTP://GITHUB.com/Path", "http://github.com/Path", false},
		{"Remove HTTP Default Port", "http://example.com:80/path", "http://example.com/path", false},
		{"Remove HTTPS Default Port", "https://example.com:443/path", "https://example.com/path", false},
		{"Keep Non-Default Port", "http://example.com:8080/path", "http://example.com:8080/path", false},
		{"Remove Trailing Root Slash", "http://example.com/", "http://example.com", false},
		{"Keep Trailing Slash in Path", "http://example.com/path/", "http://example.com/path/", false},
		{"Empty String", "", "", true},
		{"Only Spaces", "   ", "", true},
		{"No Scheme", "example.com", "", true},
		{"Invalid Scheme", "ftp://example.com", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.raw)
			if (err != nil) != tt.expectErr {
				t.Errorf("NormalizeURL() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if got != tt.expectURL {
				t.Errorf("NormalizeURL() got = %v, want %v", got, tt.expectURL)
			}
		})
	}
}

func TestGenerateCode(t *testing.T) {
	code1, err := GenerateCode()
	if err != nil {
		t.Fatalf("GenerateCode() error = %v", err)
	}
	if len(code1) != 6 {
		t.Errorf("GenerateCode() length got = %v, want 6", len(code1))
	}

	code2, _ := GenerateCode()
	if code1 == code2 {
		t.Errorf("GenerateCode() generated same code twice: %v", code1)
	}
}
