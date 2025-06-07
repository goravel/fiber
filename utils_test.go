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
