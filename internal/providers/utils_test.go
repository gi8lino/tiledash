package providers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtils_AsInt_Stringify_Trim(t *testing.T) {
	t.Parallel()

	// asInt
	assert.Equal(t, 5, asInt(5))
	assert.Equal(t, 0, asInt(-1))
	assert.Equal(t, 7, asInt(float64(7)))
	assert.Equal(t, 0, asInt(float64(-2)))
	assert.Equal(t, 3, asInt(json.Number("3")))
	assert.Equal(t, 0, asInt(json.Number("-1")))
	assert.Equal(t, 42, asInt("42"))
	assert.Equal(t, 0, asInt("nope"))
	assert.Equal(t, 0, asInt(struct{}{}))

	// stringify
	assert.Equal(t, "s", stringify("s"))
	assert.Equal(t, "9", stringify(9))
	assert.Equal(t, "8", stringify(float64(8)))
	assert.Equal(t, `{"A":1}`, stringify(struct{ A int }{A: 1}))

	// trim
	assert.Equal(t, []byte("abc"), trim([]byte("abc"), 10))
	assert.Equal(t, []byte("ab"), trim([]byte("abc"), 2))
}
