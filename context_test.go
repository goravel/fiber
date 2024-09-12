package fiber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	httpCtx := Background()
	httpCtx.WithValue("Hello", "world")
	httpCtx.WithValue("Hi", "Goravel")

	assert.Equal(t, httpCtx.Value("Hello").(string), "world")
	assert.Equal(t, httpCtx.Value("Hi").(string), "Goravel")
}
