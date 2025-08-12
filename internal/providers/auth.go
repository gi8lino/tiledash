package providers

import (
	"net/http"

	"github.com/gi8lino/tiledash/internal/config"
)

// applyAuth adds Authorization to the request using the provider's auth.
func applyAuth(r *http.Request, a *config.AuthConfig) {
	if a == nil {
		return
	}
	if a.Basic != nil {
		r.SetBasicAuth(a.Basic.Username, a.Basic.Password)
		return
	}
	if a.Bearer != nil {
		r.Header.Set("Authorization", "Bearer "+a.Bearer.Token)
	}
}
