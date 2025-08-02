package jira

import (
	"fmt"
	"net/http"
	"strings"
)

// AuthFunc is a function that modifies the HTTP request to apply authentication.
type AuthFunc func(req *http.Request)

// NewBasicAuth returns an AuthFunc that sets basic authentication headers.
func NewBasicAuth(email, token string) AuthFunc {
	return func(req *http.Request) {
		req.SetBasicAuth(strings.TrimSpace(email), strings.TrimSpace(token))
	}
}

// NewBearerAuth returns an AuthFunc that sets bearer token headers.
func NewBearerAuth(token string) AuthFunc {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
}

// ResolveAuth returns the appropriate AuthFunc based on provided credentials.
// It supports either Bearer token or Basic (email + API token) authentication.
func ResolveAuth(bearerToken, email, token string) (auth AuthFunc, method string, err error) {
	switch {
	case bearerToken != "":
		return NewBearerAuth(bearerToken), "Bearer", nil
	case email != "" && token != "":
		return NewBasicAuth(email, token), "Basic", nil
	default:
		return nil, "", fmt.Errorf("no valid auth method configured: must provide either bearer token or email+token")
	}
}
