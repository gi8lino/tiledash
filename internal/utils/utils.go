package utils

import (
	"net/http"
	"strings"

	"github.com/gi8lino/jirapanel/internal/jira"
)

// ObfuscateHeader returns an obfuscated Authorization header,
// showing only the auth scheme, first 2 and last 2 characters of the token.
// All middle characters are replaced with '*', preserving original token length.
// Example: "Basic dZ*********X1" or "Bearer ab******yz"
func ObfuscateHeader(auth string) string {
	if auth == "" {
		return ""
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 {
		return "[invalid header]"
	}

	scheme := parts[0]
	token := strings.TrimSpace(parts[1])
	n := len(token)

	if n <= 4 {
		return scheme + " " + strings.Repeat("*", n)
	}

	prefix := token[:2]
	suffix := token[n-2:]
	stars := strings.Repeat("*", n-4)

	return scheme + " " + prefix + stars + suffix
}

// GetAuthorizationHeader returns the "Authorization" header value that would be set
// by the provided AuthFunc on a dummy HTTP request.
func GetAuthorizationHeader(authFunc jira.AuthFunc) string {
	req, _ := http.NewRequest("GET", "https://dummy", nil)
	authFunc(req)
	return req.Header.Get("Authorization")
}
