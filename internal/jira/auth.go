package jira

import "fmt"

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
