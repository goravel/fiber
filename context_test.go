package fiber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	type customKeyType struct{}
	var customKey customKeyType

	httpCtx := Background()
	httpCtx.WithValue("Hello", "world")
	httpCtx.WithValue("Hi", "Goravel")
	httpCtx.WithValue(customKey, "halo")
	httpCtx.WithValue(1, "one")
	httpCtx.WithValue(2.2, "two point two")

	assert.Equal(t, "world", httpCtx.Value("Hello"))
	assert.Equal(t, "Goravel", httpCtx.Value("Hi"))
	assert.Equal(t, "halo", httpCtx.Value(customKey))
	assert.Equal(t, "one", httpCtx.Value(1))
	assert.Equal(t, "two point two", httpCtx.Value(2.2))

	ctx := httpCtx.Context()
	assert.Equal(t, "world", ctx.Value("Hello"))
	assert.Equal(t, "Goravel", ctx.Value("Hi"))
	assert.Equal(t, "halo", ctx.Value(customKey))
	assert.Equal(t, "one", ctx.Value(1))
	assert.Equal(t, "two point two", ctx.Value(2.2))
}

