package config

import (
	"os"
	"strings"
)

func AppBaseURL() string {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return strings.TrimRight(baseURL, "/")
}

func PublicURL(path string) string {
	return AppBaseURL() + "/" + strings.TrimLeft(path, "/")
}
