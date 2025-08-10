package providers

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient_SkipTLSVerify(t *testing.T) {
	t.Parallel()

	t.Run("default skip=false when nil", func(t *testing.T) {
		t.Parallel()
		c := newHTTPClient(config.Provider{})
		tr, ok := c.Transport.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, tr.TLSClientConfig)
		assert.Equal(t, false, tr.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("skipTLS true", func(t *testing.T) {
		t.Parallel()
		v := true
		c := newHTTPClient(config.Provider{SkipTLSVerify: &v})
		tr := c.Transport.(*http.Transport)
		require.NotNil(t, tr.TLSClientConfig)
		assert.Equal(t, true, tr.TLSClientConfig.InsecureSkipVerify)
	})
}

func TestNewHTTPTransport_Fields(t *testing.T) {
	t.Parallel()

	tr := newHTTPTransport(true)
	require.NotNil(t, tr)
	assert.IsType(t, &tls.Config{}, tr.TLSClientConfig)
	assert.True(t, tr.TLSClientConfig.InsecureSkipVerify)
	assert.Greater(t, tr.MaxIdleConns, 0)
	assert.Greater(t, tr.MaxIdleConnsPerHost, 0)
}

func TestDecodeJSONUseNumber(t *testing.T) {
	t.Parallel()

	t.Run("empty -> empty map", func(t *testing.T) {
		out, err := decodeJSONUseNumber(nil)
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("number preserved as json.Number", func(t *testing.T) {
		raw := []byte(`{"n":9007199254740993}`)
		out, err := decodeJSONUseNumber(raw)
		require.NoError(t, err)

		v, ok := out["n"].(json.Number)
		require.True(t, ok, "expected json.Number, got %T", out["n"])
		assert.Equal(t, "9007199254740993", v.String())
	})
}
