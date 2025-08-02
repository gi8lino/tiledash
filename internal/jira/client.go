package jira

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Searcher is an interface for Jira search API calls.
type Searcher interface {
	SearchByJQL(ctx context.Context, jql string, params map[string]string) ([]byte, int, error)
}

// Client handles communication with the Jira REST API.
type Client struct {
	APIURL *url.URL     // Base API URL (must include /rest/api/X)
	Client *http.Client // Underlying HTTP client
	auth   AuthFunc
}

// NewClient returns a Jira client with the given base URL and authentication function.
func NewClient(apiURL *url.URL, auth AuthFunc, skipVerify bool) *Client {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipVerify,
		},
	}
	return &Client{
		APIURL: apiURL,
		Client: &http.Client{Transport: tr},
		auth:   auth,
	}
}

// SearchByJQL performs a JQL search request against the Jira API.
func (c *Client) SearchByJQL(ctx context.Context, jql string, queryParams map[string]string) (response []byte, statusCode int, err error) {
	if strings.TrimSpace(jql) == "" {
		return nil, http.StatusInternalServerError, fmt.Errorf("missing JQL query")
	}

	params := url.Values{}
	params.Set("jql", jql)

	// Add additional query parameters
	for k, v := range queryParams {
		if k != "" && v != "" {
			params.Set(k, v)
		}
	}

	path := "search?" + params.Encode()
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// doRequest performs an authenticated HTTP request and returns response body, status, and error.
func (c *Client) doRequest(ctx context.Context, method, path string, body any) (response []byte, statusCode int, err error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// Parse path into relative URL with optional query
	relURL, err := url.Parse(path)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("parse path: %w", err)
	}
	fullURL := c.APIURL.ResolveReference(relURL).String()

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("create request: %w", err)
	}

	c.auth(req) // apply authentication

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("jira error: %s", string(respBody))
	}
	return respBody, resp.StatusCode, nil
}
