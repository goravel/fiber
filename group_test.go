package fiber

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
	mocksconfig "github.com/goravel/framework/mocks/config"
	"github.com/goravel/framework/support/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GroupTestSuite struct {
	suite.Suite
	mockConfig *mocksconfig.Config
	route      *Route
}

func TestGroupTestSuite(t *testing.T) {
	suite.Run(t, new(GroupTestSuite))
}

func (s *GroupTestSuite) SetupTest() {
	s.mockConfig = mocksconfig.NewConfig(s.T())
	s.mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	s.mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	s.mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	s.mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	s.mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	s.mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
	s.mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()

	route, err := NewRoute(s.mockConfig, nil)
	s.NoError(err)

	s.route = route
}

func (s *GroupTestSuite) TestGet() {
	s.route.Get("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("GET", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestPost() {
	s.route.Post("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("POST", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestPut() {
	s.route.Put("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("PUT", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestDelete() {
	s.route.Delete("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("DELETE", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestOptions() {
	s.route.Options("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("OPTIONS", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestPatch() {
	s.route.Patch("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("PATCH", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestAny() {
	s.route.Any("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("GET", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
	s.assert("POST", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
	s.assert("PUT", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
	s.assert("DELETE", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
	s.assert("OPTIONS", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
	s.assert("PATCH", "/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestResource() {
	s.route.setMiddlewares([]fiber.Handler{
		middlewareToFiberHandler(func(ctx contractshttp.Context) {
			ctx.WithValue("action", ctx.Request().Origin().Method)
			ctx.Request().Next()
		}),
	})
	s.route.Resource("/resource", resourceController{})

	s.assert("GET", "/resource", http.StatusOK, "{\"action\":\"GET\"}")
	s.assert("GET", "/resource/1", http.StatusOK, "{\"action\":\"GET\",\"id\":\"1\"}")
	s.assert("POST", "/resource", http.StatusOK, "{\"action\":\"POST\"}")
	s.assert("PUT", "/resource/1", http.StatusOK, "{\"action\":\"PUT\",\"id\":\"1\"}")
	s.assert("PATCH", "/resource/1", http.StatusOK, "{\"action\":\"PATCH\",\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestStatic() {
	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(s.T(), err)

	err = os.WriteFile(filepath.Join(tempDir, "test.json"), []byte("{\"id\":1}"), 0755)
	assert.NoError(s.T(), err)

	s.route.Static("static", tempDir)

	s.assert("GET", "/static/test.json", http.StatusOK, "{\"id\":1}")
}

func (s *GroupTestSuite) TestStaticFile() {
	file, err := os.CreateTemp("", "test")
	assert.NoError(s.T(), err)

	err = os.WriteFile(file.Name(), []byte("{\"id\":1}"), 0755)
	assert.NoError(s.T(), err)

	s.route.StaticFile("static-file", file.Name())

	s.assert("GET", "/static-file", http.StatusOK, "{\"id\":1}")
}

func (s *GroupTestSuite) TestStaticFS() {
	tempDir, err := os.MkdirTemp("", "test")
	assert.NoError(s.T(), err)

	err = os.WriteFile(filepath.Join(tempDir, "test.json"), []byte("{\"id\":1}"), 0755)
	assert.NoError(s.T(), err)

	s.route.StaticFS("static-fs", http.Dir(tempDir))

	s.assert("GET", "/static-fs/test.json", http.StatusOK, "{\"id\":1}")
}

func (s *GroupTestSuite) TestAbortMiddleware() {
	s.route.Middleware(abortMiddleware()).Get("/middleware/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("GET", "/middleware/1", http.StatusNonAuthoritativeInfo, "")
}

func (s *GroupTestSuite) TestMultipleMiddleware() {
	s.route.Middleware(contextMiddleware(), contextMiddleware1()).Get("/middlewares/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id":   ctx.Request().Input("id"),
			"ctx":  ctx.Value("ctx"),
			"ctx1": ctx.Value("ctx1"),
		})
	})

	s.assert("GET", "/middlewares/1", http.StatusOK, "{\"ctx\":\"Goravel\",\"ctx1\":\"Hello\",\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestMultiplePrefix() {
	s.route.Prefix("prefix1").Prefix("prefix2").Get("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Success().Json(contractshttp.Json{
			"id": ctx.Request().Input("id"),
		})
	})

	s.assert("GET", "/prefix1/prefix2/input/1", http.StatusOK, "{\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestMultiplePrefixGroupMiddleware() {
	s.route.Prefix("group1").Middleware(contextMiddleware()).Group(func(route1 contractsroute.Router) {
		route1.Prefix("group2").Middleware(contextMiddleware1()).Group(func(route2 contractsroute.Router) {
			route2.Get("/middleware/{id}", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Success().Json(contractshttp.Json{
					"id":   ctx.Request().Input("id"),
					"ctx":  ctx.Value("ctx"),
					"ctx1": ctx.Value("ctx1"),
				})
			})
		})
		route1.Middleware(contextMiddleware2()).Get("/middleware/{id}", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Success().Json(contractshttp.Json{
				"id":   ctx.Request().Input("id"),
				"ctx":  ctx.Value("ctx"),
				"ctx2": ctx.Value("ctx2"),
			})
		})
	})

	s.assert("GET", "/group1/group2/middleware/1", http.StatusOK, "{\"ctx\":\"Goravel\",\"ctx1\":\"Hello\",\"id\":\"1\"}")
	s.assert("GET", "/group1/middleware/1", http.StatusOK, "{\"ctx\":\"Goravel\",\"ctx2\":\"World\",\"id\":\"1\"}")
}

func (s *GroupTestSuite) TestGlobalMiddleware() {
	s.mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
	s.mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()
	s.mockConfig.EXPECT().Get("cors.paths").Return([]string{"*"}).Once()
	s.mockConfig.EXPECT().Get("cors.allowed_methods").Return([]string{"*"}).Once()
	s.mockConfig.EXPECT().Get("cors.allowed_origins").Return([]string{"*"}).Once()
	s.mockConfig.EXPECT().Get("cors.allowed_headers").Return([]string{"*"}).Once()
	s.mockConfig.EXPECT().Get("cors.exposed_headers").Return([]string{"*"}).Once()
	s.mockConfig.EXPECT().GetInt("cors.max_age").Return(0).Once()
	s.mockConfig.EXPECT().GetBool("cors.supports_credentials").Return(false).Once()
	s.mockConfig.EXPECT().GetInt("http.request_timeout", 3).Return(1).Once()
	ConfigFacade = s.mockConfig

	s.route.GlobalMiddleware(func(ctx contractshttp.Context) {
		ctx.WithValue("global", "goravel")
		ctx.Request().Next()
	})
	s.route.Get("/global-middleware", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(http.StatusOK, contractshttp.Json{
			"global": ctx.Value("global"),
		})
	})

	s.assert("GET", "/global-middleware", http.StatusOK, "{\"global\":\"goravel\"}")
}

func (s *GroupTestSuite) TestMiddlewareConflict() {
	s.route.Prefix("conflict").Group(func(route1 contractsroute.Router) {
		route1.Middleware(contextMiddleware()).Get("/middleware1/{id}", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Success().Json(contractshttp.Json{
				"id":   ctx.Request().Input("id"),
				"ctx":  ctx.Value("ctx"),
				"ctx2": ctx.Value("ctx2"),
			})
		})
		route1.Middleware(contextMiddleware2()).Post("/middleware2/{id}", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Success().Json(contractshttp.Json{
				"id":   ctx.Request().Input("id"),
				"ctx":  ctx.Value("ctx"),
				"ctx2": ctx.Value("ctx2"),
			})
		})
	})

	s.assert("POST", "/conflict/middleware2/1", http.StatusOK, "{\"ctx\":null,\"ctx2\":\"World\",\"id\":\"1\"}")
}

// https://github.com/goravel/goravel/issues/408
func (s *GroupTestSuite) TestIssue408() {
	s.route.Prefix("prefix/{id}").Group(func(route contractsroute.Router) {
		route.Get("", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().String(200, "ok")
		})
		route.Post("test/{name}", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().String(200, "ok")
		})
	})

	routes := s.route.GetRoutes()
	s.Len(routes, 3)
	s.Equal("GET", routes[0].Method)
	s.Equal("/prefix/{id}", routes[0].Path)
	s.Equal("HEAD", routes[1].Method)
	s.Equal("/prefix/{id}", routes[1].Path)
	s.Equal("POST", routes[2].Method)
	s.Equal("/prefix/{id}/test/{name}", routes[2].Path)
}

func (s *GroupTestSuite) assert(method, url string, expectCode int, expectBody string) {
	req, err := http.NewRequest(method, url, nil)
	s.Nil(err)

	resp, err := s.route.Test(req)
	s.NoError(err)

	s.Equal(expectCode, resp.StatusCode)

	if expectBody != "" {
		body, err := io.ReadAll(resp.Body)
		s.NoError(err)

		bodyMap := make(map[string]any)
		exceptBodyMap := make(map[string]any)

		err = json.Unmarshal(body, &bodyMap)
		s.NoError(err)
		err = json.UnmarshalString(expectBody, &exceptBodyMap)
		s.NoError(err)

		s.Equal(exceptBodyMap, bodyMap)
	}
}

func abortMiddleware() contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		ctx.Request().Abort(http.StatusNonAuthoritativeInfo)
	}
}

func contextMiddleware() contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		ctx.WithValue("ctx", "Goravel")

		ctx.Request().Next()
	}
}

func contextMiddleware1() contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		ctx.WithValue("ctx1", "Hello")

		ctx.Request().Next()
	}
}

func contextMiddleware2() contractshttp.Middleware {
	return func(ctx contractshttp.Context) {
		ctx.WithValue("ctx2", "World")

		ctx.Request().Next()
	}
}

type resourceController struct{}

func (c resourceController) Index(ctx contractshttp.Context) contractshttp.Response {
	action := ctx.Value("action")

	return ctx.Response().Json(http.StatusOK, contractshttp.Json{
		"action": action,
	})
}

func (c resourceController) Show(ctx contractshttp.Context) contractshttp.Response {
	action := ctx.Value("action")
	id := ctx.Request().Input("id")

	return ctx.Response().Json(http.StatusOK, contractshttp.Json{
		"action": action,
		"id":     id,
	})
}

func (c resourceController) Store(ctx contractshttp.Context) contractshttp.Response {
	action := ctx.Value("action")

	return ctx.Response().Json(http.StatusOK, contractshttp.Json{
		"action": action,
	})
}

func (c resourceController) Update(ctx contractshttp.Context) contractshttp.Response {
	action := ctx.Value("action")
	id := ctx.Request().Input("id")

	return ctx.Response().Json(http.StatusOK, contractshttp.Json{
		"action": action,
		"id":     id,
	})
}

func (c resourceController) Destroy(ctx contractshttp.Context) contractshttp.Response {
	action := ctx.Value("action")
	id := ctx.Request().Input("id")

	return ctx.Response().Json(http.StatusOK, contractshttp.Json{
		"action": action,
		"id":     id,
	})
}
