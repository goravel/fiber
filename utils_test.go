package fiber

import (
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
)

func TestBracketToColon(t *testing.T) {
	assert.Equal(t, "/:id/:name", bracketToColon("/{id}/{name}"))
}

func TestColonToBracket(t *testing.T) {
	assert.Equal(t, "/{id}/{name}", colonToBracket("/:id/:name"))
}

func TestIsSameMiddleware(t *testing.T) {
	mw1 := &testMiddleware{id: "foo"}
	mw2 := &testMiddleware{id: "foo"}
	mw3 := &testMiddleware{id: "bar"}

	assert.True(t, isSameMiddleware(mw1, mw2))
	assert.False(t, isSameMiddleware(mw1, mw3))
	assert.False(t, isSameMiddleware(nil, mw1))
	assert.False(t, isSameMiddleware("not middleware", mw1))
}

type testMiddleware struct{ id string }

func (m *testMiddleware) Handle(contractshttp.Context) {}

func (m *testMiddleware) Signature() string { return m.id }
