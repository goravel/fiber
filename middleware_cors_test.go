package fiber

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"

	contractshttp "github.com/goravel/framework/contracts/http"
	configmocks "github.com/goravel/framework/mocks/config"
	"github.com/stretchr/testify/assert"
)

func TestCors(t *testing.T) {
	var (
		mockConfig *configmocks.Config
		resp       *http.Response
	)
	beforeEach := func() {
		mockConfig = configmocks.NewConfig(t)
	}

	tests := []struct {
		name   string
		setup  func()
		assert func()
	}{
		{
			name: "allow all paths",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "GET,POST,HEAD,PUT,DELETE,PATCH", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "not allow path",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"api"}).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "allow path with *",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"any/*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "GET,POST,HEAD,PUT,DELETE,PATCH", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "only allow POST",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"POST"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "POST", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "not allow POST",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"GET"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "GET", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "not allow origin",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"https://goravel.com"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "GET,POST,HEAD,PUT,DELETE,PATCH", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "allow specific origin",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"https://www.goravel.dev"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "GET,POST,HEAD,PUT,DELETE,PATCH", resp.Header.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "https://www.goravel.dev", resp.Header.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "", resp.Header.Get("Access-Control-Expose-Headers"))
			},
		},
		{
			name: "not allow exposed headers",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"Goravel"}).Once()
				mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
				mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig
			},
			assert: func() {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
				assert.Equal(t, "GET,POST,HEAD,PUT,DELETE,PATCH", resp.Header.Get("Access-Control-Allow-Methods"))
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

			route, err := NewRoute(mockConfig, nil)
			assert.Nil(t, err)
			route.setMiddlewares([]fiber.Handler{
				middlewareToFiberHandler(Cors()),
			})
			route.Post("/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Success().Json(contractshttp.Json{
					"id": ctx.Request().Input("id"),
				})
			})

			req, err := http.NewRequest("OPTIONS", "/any/1", nil)
			assert.Nil(t, err)

			req.Header.Set("Origin", "https://www.goravel.dev")
			req.Header.Set("Access-Control-Request-Method", "POST")

			resp, err = route.Test(req)
			assert.NoError(t, err, test.name)

			test.assert()
		})
	}
}
