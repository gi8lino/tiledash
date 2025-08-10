package providers

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/gi8lino/tiledash/internal/config"
)

// newHTTPTransport returns a tuned Transport with optional TLS skipping.
func newHTTPTransport(skipInsecure bool) *http.Transport {
	// use sane pooling so pagination isn’t penalized
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,

		// reasonable connection pooling defaults
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,

		// TCP settings
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,

		// TLS (respect config)
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipInsecure, // NOTE: intended for dev only
		},

		// timeouts on TLS handshake / expect-continue can help with slow remotes
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// newHTTPClient builds an http.Client with transport + request timeout.
func newHTTPClient(pc config.Provider) *http.Client {
	skip := false
	if pc.SkipTLSVerify != nil {
		skip = *pc.SkipTLSVerify
	}
	return &http.Client{
		Timeout:   15 * time.Second,       // hard per-request cap
		Transport: newHTTPTransport(skip), // pooled transport
	}
}
