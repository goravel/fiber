package fiber

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	type customType struct{}
	var customTypeKey customType

	httpCtx := Background()
	httpCtx.WithValue("Hello", "world")
	httpCtx.WithValue("Hi", "Goravel")
	httpCtx.WithValue(customTypeKey, "halo")

	userContext := context.Background()
	userContext = context.WithValue(userContext, "user_a", "b")
	httpCtx.WithContext(userContext)

	httpCtx.WithValue(1, "one")
	httpCtx.WithValue(2.2, "two point two")

	assert.Equal(t, "world", httpCtx.Value("Hello"))
	assert.Equal(t, "Goravel", httpCtx.Value("Hi"))
	assert.Equal(t, "halo", httpCtx.Value(customTypeKey))
	assert.Equal(t, "one", httpCtx.Value(1))
	assert.Equal(t, "two point two", httpCtx.Value(2.2))
	assert.NotNil(t, httpCtx.Value(contextKey))

	// The value of UserContext can't be covered.
	assert.Equal(t, "b", httpCtx.Value("user_a"))

	ctx := httpCtx.Context()
	assert.Equal(t, "world", ctx.Value("Hello"))
	assert.Equal(t, "Goravel", ctx.Value("Hi"))
	assert.Equal(t, "halo", ctx.Value(customTypeKey))
	assert.Equal(t, "b", ctx.Value("user_a"))
	assert.Equal(t, "one", ctx.Value(1))
	assert.Equal(t, "two point two", ctx.Value(2.2))
	assert.Nil(t, ctx.Value(contextKey))
}
