package main

import (
	"os"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	os.Setenv("TEST_KEY", "value")
	defer os.Unsetenv("TEST_KEY")

	if got := getEnvOrDefault("TEST_KEY", "fallback"); got != "value" {
		t.Errorf("expected value, got %s", got)
	}

	if got := getEnvOrDefault("NON_EXISTENT_TEST_KEY", "fallback"); got != "fallback" {
		t.Errorf("expected fallback, got %s", got)
	}
}
