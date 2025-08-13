package utils

import (
	"net/url"
	"strings"
)

// NormalizeRoutePrefix returns "" or "/prefix" from input, accepting raw paths or full URLs.
func NormalizeRoutePrefix(input string) string {
	s := strings.TrimSpace(input)
	if s == "" || s == "/" {
		return ""
	}
	// If someone passes a full URL, keep only the .Path.
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil {
			s = u.Path
		}
	}
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "/")
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	if s == "/" {
		return ""
	}
	return s
}

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

// Ptr returns a pointer to the given value.
func Ptr[T any](v T) *T { return &v }
