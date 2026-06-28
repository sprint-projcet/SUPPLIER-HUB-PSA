package config

import (
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var GoogleOAuthConfig *oauth2.Config

func SetupGoogleOAuth() {
	redirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = PublicURL("/api/auth/google/callback")
	}

	GoogleOAuthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func IsGoogleOAuthConfigured() bool {
	return GoogleOAuthConfig != nil &&
		GoogleOAuthConfig.ClientID != "" &&
		GoogleOAuthConfig.ClientSecret != "" &&
		GoogleOAuthConfig.RedirectURL != ""
}
