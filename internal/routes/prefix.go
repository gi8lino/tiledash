package routes

import (
	"net/http"
	"net/url"
	"strings"
)

type prefixRedirectHandler struct {
	prefix string
}

func (h prefixRedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlePrefixRedirect(w, r, h.prefix)
}

// NormalizeRoutePrefix returns "" or "/prefix" from input,
// accepting raw paths or full URLs.
func NormalizeRoutePrefix(input string) string {
	s := strings.TrimSpace(input)
	if s == "" || s == "/" {
		return ""
	}

	// Attempt URL parse. Only treat it as a URL if a scheme is present.
	if u, err := url.Parse(s); err == nil && u.Scheme != "" {
		s = u.Path
	}

	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "/")

	if s == "" || s == "/" {
		return ""
	}

	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}

	return s
}

// mountUnderPrefix mounts h under the given route prefix and redirects bare prefix to prefix/.
func mountUnderPrefix(h http.Handler, prefix string) http.Handler {
	// Normalize.
	if prefix == "" || prefix == "/" {
		return h
	}
	if prefix[0] != '/' {
		prefix = "/" + prefix
	}
	// Trim trailing slashes.
	for len(prefix) > 1 && prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}

	mux := http.NewServeMux()

	// Redirect bare "/tambua" -> "/tambua/" so subtree handlers match.
	// Use a path-only pattern so behavior stays consistent across Go versions.
	mux.Handle(prefix, prefixRedirectHandler{prefix: prefix})

	// Mount everything under prefix and strip it so internal routes live at "/".
	mux.Handle(prefix+"/", http.StripPrefix(prefix, h))

	return mux
}

// handlePrefixRedirect redirects bare prefix requests to the slash-suffixed path.
func handlePrefixRedirect(w http.ResponseWriter, r *http.Request, prefix string) {
	status := http.StatusTemporaryRedirect // 307 (preserve method/body)
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		status = http.StatusPermanentRedirect // 308
	}
	http.Redirect(w, r, prefix+"/", status)
}
