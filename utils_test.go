package fiber

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBracketToColon(t *testing.T) {
	assert.Equal(t, "/:id/:name", bracketToColon("/{id}/{name}"))
}

func TestColonToBracket(t *testing.T) {
	assert.Equal(t, "/{id}/{name}", colonToBracket("/:id/:name"))
}

func TestIsSameMiddleware(t *testing.T) {
	type mwA struct{}
	type mwB struct{}

	assert.True(t, isSameMiddleware(&mwA{}, &mwA{}))
	assert.False(t, isSameMiddleware(&mwA{}, &mwB{}))
	fn1 := func() {}
	fn2 := func() {}
	assert.True(t, isSameMiddleware(fn1, fn2))
	assert.False(t, isSameMiddleware(nil, fn1))
}
