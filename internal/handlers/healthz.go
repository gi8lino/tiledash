package handlers

import "net/http"

// Healthz handles the /healthz endpoint
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok")) // nolint:errcheck
	}
}
