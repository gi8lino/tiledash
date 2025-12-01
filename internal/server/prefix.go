package server

import (
	"net/http"
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

// mountUnderPrefix mounts h under the given route prefix, adding a redirect from bare prefix → prefix/.
func mountUnderPrefix(h http.Handler, prefix string) http.Handler {
	if prefix == "" {
		return h // serve at root
	}
	mux := http.NewServeMux()

	// Mount everything under prefix and strip it so internal routes live at "/".
	// ServeMux will automatically redirect bare "/tiledash" to "/tiledash/" because
	// the registered pattern ends with a slash.
	mux.Handle(prefix+"/", http.StripPrefix(prefix, h))

	// Not mounting at "/" ensures non-prefixed URLs 404, which is desirable when hosting under a subpath.
	return mux
}
