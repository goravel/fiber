package fiber

import (
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
)

func TestNewAction(t *testing.T) {
	// Clear routes map before test
	routes = make(map[string]map[string]contractshttp.Info)

	// Test creating a new action
	action := NewAction("GET", "/test-path", "test.Action")
	assert.NotNil(t, action)
	assert.IsType(t, &Action{}, action)

	// Verify route was added to routes map
	routeInfo, exists := routes["/test-path"]["GET|HEAD"]
	assert.True(t, exists)
	assert.Equal(t, "GET|HEAD", routeInfo.Method)
	assert.Equal(t, "/test-path", routeInfo.Path)
	assert.Equal(t, "test.Action", routeInfo.Handler)
	assert.Empty(t, routeInfo.Name)
}

func TestAction_Name(t *testing.T) {
	// Clear routes map before test
	routes = make(map[string]map[string]contractshttp.Info)

	// Create a new action
	action := NewAction("GET", "/named-path", "")

	// Test setting name
	namedAction := action.Name("test-route")
	assert.NotNil(t, namedAction)
	assert.IsType(t, &Action{}, namedAction)

	// Verify route info was updated
	routeInfo, exists := routes["/named-path"]["GET|HEAD"]
	assert.True(t, exists)
	assert.Equal(t, "GET|HEAD", routeInfo.Method)
	assert.Equal(t, "/named-path", routeInfo.Path)
	assert.Equal(t, "test-route", routeInfo.Name)

	// Test method chaining
	chainedAction := action.Name("new-name").Name("final-name")
	assert.NotNil(t, chainedAction)

	// Verify final route info
	routeInfo, exists = routes["/named-path"]["GET|HEAD"]
	assert.True(t, exists)
	assert.Equal(t, "final-name", routeInfo.Name)
}
