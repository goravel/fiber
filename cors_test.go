package fiber

import (
	"net/http"
	"testing"

	configmocks "github.com/goravel/framework/contracts/config/mocks"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	var (
		mockConfig *configmocks.Config
		resp       *http.Response
	)
	beforeEach := func() {
		mockConfig = &configmocks.Config{}
	}

	tests := []struct {
		name   string
		setup  func()
		assert func()
	}{
		{
			name: "allow all paths",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "not allow path",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"api"}).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "allow path with *",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"any/*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "allow POST",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"GET"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "allow origin",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"goravel.dev"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "not allow exposed headers",
			setup: func() {
				mockConfig.On("GetString", "app.name", "Goravel").Return("Goravel").Once()
				mockConfig.On("GetBool", "app.debug", false).Return(true).Twice()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"Goravel"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "Goravel", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			test.setup()

			f := NewRoute(mockConfig)
			f.Any("/any/{id}", func(ctx contractshttp.Context) {
				ctx.Response().Success().Json(contractshttp.Json{
					"id": ctx.Request().Input("id"),
				})
			})

			req, err := http.NewRequest("POST", "/any/1", nil)
			assert.Nil(t, err)
			req.Header.Set("Origin", "http://127.0.0.1")

			resp, err = f.Test(req)
			assert.NoError(t, err, test.name)

			test.assert()

			mockConfig.AssertExpectations(t)
		})
	}
}
