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
)

func TestGroup(t *testing.T) {
	var (
		route      *Route
		mockConfig *mocksconfig.Config
	)
	beforeEach := func() {
		mockConfig = mocksconfig.NewConfig(t)
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
		mockConfig.On("GetBool", "http.drivers.fiber.immutable", true).Return(true).Once()
		mockConfig.On("GetInt", "http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.On("GetInt", "http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
		mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
		ConfigFacade = mockConfig
	}
	tests := []struct {
		name           string
		setup          func(req *http.Request)
		method         string
		url            string
		expectCode     int
		expectBodyJson string
	}{
		{
			name: "Get",
			setup: func(req *http.Request) {
				route.Get("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Json(http.StatusOK, contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "GET",
			url:            "/input/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Post",
			setup: func(req *http.Request) {
				route.Post("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "POST",
			url:            "/input/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Put",
			setup: func(req *http.Request) {
				route.Put("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "PUT",
			url:            "/input/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Delete",
			setup: func(req *http.Request) {
				route.Delete("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "DELETE",
			url:            "/input/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Options",
			setup: func(req *http.Request) {
				route.Options("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:     "OPTIONS",
			url:        "/input/1",
			expectCode: http.StatusOK,
		},
		{
			name: "Patch",
			setup: func(req *http.Request) {
				route.Patch("/input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "PATCH",
			url:            "/input/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Any Get",
			setup: func(req *http.Request) {
				route.Any("/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "GET",
			url:            "/any/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Any Post",
			setup: func(req *http.Request) {
				route.Any("/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "POST",
			url:            "/any/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Any Put",
			setup: func(req *http.Request) {
				route.Any("/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "PUT",
			url:            "/any/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Any Delete",
			setup: func(req *http.Request) {
				route.Any("/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "DELETE",
			url:            "/any/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Any Patch",
			setup: func(req *http.Request) {
				route.Any("/any/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "PATCH",
			url:            "/any/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Resource Index",
			setup: func(req *http.Request) {
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.WithValue("action", "index")
						ctx.Request().Next()
					}),
				})
				route.Resource("/resource", resourceController{})
			},
			method:         "GET",
			url:            "/resource",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"index\"}",
		},
		{
			name: "Resource Show",
			setup: func(req *http.Request) {
				resource := resourceController{}
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.WithValue("action", "show")
						ctx.Request().Next()
					}),
				})
				route.Resource("/resource", resource)
			},
			method:         "GET",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"show\",\"id\":\"1\"}",
		},
		{
			name: "Resource Store",
			setup: func(req *http.Request) {
				resource := resourceController{}
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.WithValue("action", "store")
						ctx.Request().Next()
					}),
				})
				route.Resource("/resource", resource)
			},
			method:         "POST",
			url:            "/resource",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"store\"}",
		},
		{
			name: "Resource Update (PUT)",
			setup: func(req *http.Request) {
				resource := resourceController{}
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.WithValue("action", "update")
						ctx.Request().Next()
					}),
				})
				route.Resource("/resource", resource)
			},
			method:         "PUT",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"update\",\"id\":\"1\"}",
		},
		{
			name: "Resource Update (PATCH)",
			setup: func(req *http.Request) {
				resource := resourceController{}
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.WithValue("action", "update")
						ctx.Request().Next()
					}),
				})
				route.Resource("/resource", resource)
			},
			method:         "PATCH",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"update\",\"id\":\"1\"}",
		},
		{
			name: "Resource Destroy",
			setup: func(req *http.Request) {
				resource := resourceController{}
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.WithValue("action", "destroy")
						ctx.Request().Next()
					}),
				})
				route.Resource("/resource", resource)
			},
			method:         "DELETE",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"destroy\",\"id\":\"1\"}",
		},
		{
			name: "Static",
			setup: func(req *http.Request) {
				tempDir, err := os.MkdirTemp("", "test")
				assert.NoError(t, err)

				err = os.WriteFile(filepath.Join(tempDir, "test.json"), []byte("{\"id\":1}"), 0755)
				assert.NoError(t, err)

				route.Static("static", tempDir)
			},
			method:         "GET",
			url:            "/static/test.json",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":1}",
		},
		{
			name: "StaticFile",
			setup: func(req *http.Request) {
				file, err := os.CreateTemp("", "test")
				assert.NoError(t, err)

				err = os.WriteFile(file.Name(), []byte("{\"id\":1}"), 0755)
				assert.NoError(t, err)

				route.StaticFile("static-file", file.Name())
			},
			method:         "GET",
			url:            "/static-file",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":1}",
		},
		{
			name: "StaticFS",
			setup: func(req *http.Request) {
				tempDir, err := os.MkdirTemp("", "test")
				assert.NoError(t, err)

				err = os.WriteFile(filepath.Join(tempDir, "test.json"), []byte("{\"id\":1}"), 0755)
				assert.NoError(t, err)

				route.StaticFS("static-fs", http.Dir(tempDir))
			},
			method:         "GET",
			url:            "/static-fs/test.json",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":1}",
		},
		{
			name: "Abort Middleware",
			setup: func(req *http.Request) {
				route.Middleware(abortMiddleware()).Get("/middleware/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:     "GET",
			url:        "/middleware/1",
			expectCode: http.StatusNonAuthoritativeInfo,
		},
		{
			name: "Multiple Middleware",
			setup: func(req *http.Request) {
				route.Middleware(contextMiddleware(), contextMiddleware1()).Get("/middlewares/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id":   ctx.Request().Input("id"),
						"ctx":  ctx.Value("ctx"),
						"ctx1": ctx.Value("ctx1"),
					})
				})
			},
			method:         "GET",
			url:            "/middlewares/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"ctx\":\"Goravel\",\"ctx1\":\"Hello\",\"id\":\"1\"}",
		},
		{
			name: "Multiple Prefix",
			setup: func(req *http.Request) {
				route.Prefix("prefix1").Prefix("prefix2").Get("input/{id}", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
						"id": ctx.Request().Input("id"),
					})
				})
			},
			method:         "GET",
			url:            "/prefix1/prefix2/input/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name: "Multiple Prefix Group Middleware",
			setup: func(req *http.Request) {
				route.Prefix("group1").Middleware(contextMiddleware()).Group(func(route1 contractsroute.Router) {
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
			},
			method:         "GET",
			url:            "/group1/group2/middleware/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"ctx\":\"Goravel\",\"ctx1\":\"Hello\",\"id\":\"1\"}",
		},
		{
			name: "Multiple Group Middleware",
			setup: func(req *http.Request) {
				route.Prefix("group1").Middleware(contextMiddleware()).Group(func(route1 contractsroute.Router) {
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
			},
			method:         "GET",
			url:            "/group1/middleware/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"ctx\":\"Goravel\",\"ctx2\":\"World\",\"id\":\"1\"}",
		},
		{
			name: "Global Middleware",
			setup: func(req *http.Request) {
				mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				mockConfig.On("GetInt", "http.request_timeout", 3).Return(1).Once()

				route.GlobalMiddleware(func(ctx contractshttp.Context) {
					ctx.WithValue("global", "goravel")
					ctx.Request().Next()
				})
				route.Get("/global-middleware", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Json(http.StatusOK, contractshttp.Json{
						"global": ctx.Value("global"),
					})
				})
			},
			method:         "GET",
			url:            "/global-middleware",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"global\":\"goravel\"}",
		},
		{
			name: "Middleware Conflict",
			setup: func(req *http.Request) {
				route.Prefix("conflict").Group(func(route1 contractsroute.Router) {
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
			},
			method:         "POST",
			url:            "/conflict/middleware2/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"ctx\":null,\"ctx2\":\"World\",\"id\":\"1\"}",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			req, err := http.NewRequest(test.method, test.url, nil)
			assert.Nil(t, err)

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			if test.setup != nil {
				test.setup(req)
			}

			resp, err := route.Test(req)
			assert.NoError(t, err, test.name)

			if test.expectBodyJson != "" {
				body, _ := io.ReadAll(resp.Body)
				bodyMap := make(map[string]any)
				exceptBodyMap := make(map[string]any)

				err = json.Unmarshal(body, &bodyMap)
				assert.NoError(t, err, test.name)
				err = json.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.NoError(t, err, test.name)

				assert.Equal(t, exceptBodyMap, bodyMap, test.name)
			}

			assert.Equal(t, test.expectCode, resp.StatusCode, test.name)
		})
	}
}

// https://github.com/goravel/goravel/issues/408
func TestIssue408(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.On("GetBool", "http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.On("GetInt", "http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.On("GetInt", "http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()

	route, err := NewRoute(mockConfig, nil)
	assert.Nil(t, err)

	route.Prefix("prefix/{id}").Group(func(route contractsroute.Router) {
		route.Get("", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().String(200, "ok")
		})
		route.Post("test/{name}", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().String(200, "ok")
		})
	})

	routes := route.GetRoutes()
	assert.Len(t, routes, 3)
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "/prefix/{id}", routes[0].Path)
	assert.Equal(t, "HEAD", routes[1].Method)
	assert.Equal(t, "/prefix/{id}", routes[1].Path)
	assert.Equal(t, "POST", routes[2].Method)
	assert.Equal(t, "/prefix/{id}/test/{name}", routes[2].Path)
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
