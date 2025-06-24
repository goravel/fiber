package fiber

import (
	"testing"

	contractsroute "github.com/goravel/framework/contracts/route"
	"github.com/stretchr/testify/assert"
)

func TestNewAction(t *testing.T) {
	// Clear routes map before test
	routes = make(map[string]map[string]contractsroute.Info)

	// Test creating a new action
	action := NewAction("GET", "/test-path", "test.Action")
	assert.NotNil(t, action)
	assert.IsType(t, &Action{}, action)

	// Verify route was added to routes map
	routeInfo, exists := routes["/test-path"]["GET"]
	assert.True(t, exists)
	assert.Equal(t, "GET", routeInfo.Method)
	assert.Equal(t, "/test-path", routeInfo.Path)
	assert.Equal(t, "test.Action", routeInfo.Handler)
	assert.Empty(t, routeInfo.Name)
}

func TestAction_Name(t *testing.T) {
	// Clear routes map before test
	routes = make(map[string]map[string]contractsroute.Info)

	// Create a new action
	action := NewAction("POST", "/named-path", "")

	// Test setting name
	namedAction := action.Name("test-route")
	assert.NotNil(t, namedAction)
	assert.IsType(t, &Action{}, namedAction)

	// Verify route info was updated
	routeInfo, exists := routes["/named-path"]["POST"]
	assert.True(t, exists)
	assert.Equal(t, "POST", routeInfo.Method)
	assert.Equal(t, "/named-path", routeInfo.Path)
	assert.Equal(t, "test-route", routeInfo.Name)

	// Test method chaining
	chainedAction := action.Name("new-name").Name("final-name")
	assert.NotNil(t, chainedAction)

	// Verify final route info
	routeInfo, exists = routes["/named-path"]["POST"]
	assert.True(t, exists)
	assert.Equal(t, "final-name", routeInfo.Name)
}
