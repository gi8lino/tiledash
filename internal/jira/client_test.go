package jira

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("creates a new client with given parameters", func(t *testing.T) {
		t.Parallel()

		rawURL := "https://jira.example.com/rest/api/2/"
		parsed, err := url.Parse(rawURL)
		require.NoError(t, err)

		auth := func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer dummy")
		}

		client := NewClient(parsed, auth, true, 2*time.Second)

		assert.Equal(t, parsed, client.APIURL)
		assert.NotNil(t, client.Client)
		assert.NotNil(t, client.auth)
	})
}

func TestSearchByJQL(t *testing.T) {
	t.Parallel()

	t.Run("missing JQL returns error", func(t *testing.T) {
		t.Parallel()

		c := &Client{}
		resp, code, err := c.SearchByJQL(context.Background(), "   ", nil)

		assert.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Nil(t, resp)
	})

	t.Run("empty JQL returns error", func(t *testing.T) {
		t.Parallel()

		server := NewClient(&url.URL{}, func(r *http.Request) {}, false, 2*time.Second)
		resp, code, err := server.SearchByJQL(context.Background(), "", nil)

		assert.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Nil(t, resp)
	})

	t.Run("filters out empty query params", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "project = TEST", r.URL.Query().Get("jql"))
			assert.Equal(t, "names", r.URL.Query().Get("expand"))
			assert.Empty(t, r.URL.Query().Get("empty"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`)) // nolint:errcheck
		}))
		defer srv.Close()

		apiURL, _ := url.Parse(srv.URL + "/rest/api/2/")
		client := &Client{
			APIURL: apiURL,
			Client: srv.Client(),
			auth:   func(r *http.Request) {},
		}

		_, _, err := client.SearchByJQL(context.Background(), "project = TEST", map[string]string{
			"expand": "names",
			"empty":  "",
		})
		assert.NoError(t, err)
	})

	t.Run("calls doRequest with correct method and path", func(t *testing.T) {
		t.Parallel()

		called := false
		client := &Client{
			APIURL: mustParseURL(t, "https://jira.example.com/rest/api/2/"),
			Client: &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					called = true
					assert.Equal(t, "GET", r.Method)
					assert.Contains(t, r.URL.String(), "search?jql=project+%3D+TEST")
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"result":true}`)),
					}, nil
				}),
			},
			auth: func(r *http.Request) {},
		}

		resp, code, err := client.SearchByJQL(context.Background(), "project = TEST", nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Contains(t, string(resp), "result")
		assert.True(t, called)
	})
}

func TestDoRequest(t *testing.T) {
	t.Parallel()

	t.Run("returns error for invalid URL path", func(t *testing.T) {
		t.Parallel()

		c := NewClient(mustParseURL(t, "https://example.com"), func(r *http.Request) {}, false, 2*time.Second)
		_, code, err := c.doRequest(context.Background(), http.MethodGet, "%%%", nil)

		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, err.Error(), "parse path")
	})

	t.Run("marshals body and sets reader", func(t *testing.T) {
		t.Parallel()

		var gotBody string

		client := &Client{
			APIURL: mustParseURL(t, "https://example.com"),
			Client: &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					b, _ := io.ReadAll(r.Body)
					gotBody = string(b)
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
					}, nil
				}),
			},
			auth: func(r *http.Request) {},
		}

		_, code, err := client.doRequest(context.Background(), http.MethodPost, "foo", map[string]string{"key": "value"})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Contains(t, gotBody, `"key":"value"`)
	})

	t.Run("returns error on marshaling failure", func(t *testing.T) {
		t.Parallel()

		client := NewClient(mustParseURL(t, "https://example.com"), func(r *http.Request) {}, false, 2*time.Second)
		_, code, err := client.doRequest(context.Background(), http.MethodPost, "foo", func() {}) // unmarshalable

		assert.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, code)
		assert.Contains(t, err.Error(), "marshal body")
	})

	t.Run("returns error on client.Do failure", func(t *testing.T) {
		t.Parallel()

		client := &Client{
			APIURL: mustParseURL(t, "https://example.com"),
			Client: &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return nil, errors.New("connection refused")
				}),
			},
			auth: func(r *http.Request) {},
		}

		_, code, err := client.doRequest(context.Background(), http.MethodGet, "foo", nil)

		assert.Error(t, err)
		assert.Equal(t, http.StatusBadGateway, code)
		assert.Contains(t, err.Error(), "do request")
	})

	t.Run("returns error on non-2xx response", func(t *testing.T) {
		t.Parallel()

		client := &Client{
			APIURL: mustParseURL(t, "https://example.com"),
			Client: &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 404,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
					}, nil
				}),
			},
			auth: func(r *http.Request) {},
		}

		body, code, err := client.doRequest(context.Background(), http.MethodGet, "foo", nil)

		assert.Error(t, err)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "not found", string(body))
	})

	t.Run("reads and returns valid response", func(t *testing.T) {
		t.Parallel()

		client := &Client{
			APIURL: mustParseURL(t, "https://example.com"),
			Client: &http.Client{
				Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
					}, nil
				}),
			},
			auth: func(r *http.Request) {},
		}

		body, code, err := client.doRequest(context.Background(), http.MethodGet, "foo", nil)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.JSONEq(t, `{"ok":true}`, string(body))
	})
}

func TestDoRequest_ReadBodyFailure(t *testing.T) {
	t.Parallel()

	apiURL := mustParseURL(t, "https://example.com/api/")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(brokenReader{}),
	}

	client := &Client{
		APIURL: apiURL,
		Client: &http.Client{
			Transport: mockDoer{resp: resp},
		},
		auth: func(r *http.Request) {},
	}

	_, code, err := client.doRequest(context.Background(), "GET", "search", nil)

	assert.Error(t, err)
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, err.Error(), "read response")
}

// brokenReader always fails
type brokenReader struct{}

func (brokenReader) Read(p []byte) (int, error) { return 0, errors.New("fail") }
func (brokenReader) Close() error               { return nil }

type mockDoer struct {
	resp *http.Response
	err  error
}

func (m mockDoer) RoundTrip(r *http.Request) (*http.Response, error) { return m.resp, m.err }

type roundTripperFunc func(r *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
