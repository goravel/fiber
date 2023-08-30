package fiber

import (
	"io"
	"net/http"
	"testing"

	"github.com/bytedance/sonic"
	configmock "github.com/goravel/framework/contracts/config/mocks"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/route"
	"github.com/stretchr/testify/assert"
)

type resourceController struct{}

func (c resourceController) Index(ctx httpcontract.Context) {
	action := ctx.Value("action")
	ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"action": action,
	})
}

func (c resourceController) Show(ctx httpcontract.Context) {
	action := ctx.Value("action")
	id := ctx.Request().Input("id")
	ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"action": action,
		"id":     id,
	})
}

func (c resourceController) Store(ctx httpcontract.Context) {
	action := ctx.Value("action")
	ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"action": action,
	})
}

func (c resourceController) Update(ctx httpcontract.Context) {
	action := ctx.Value("action")
	id := ctx.Request().Input("id")
	ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"action": action,
		"id":     id,
	})
}

func (c resourceController) Destroy(ctx httpcontract.Context) {
	action := ctx.Value("action")
	id := ctx.Request().Input("id")
	ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"action": action,
		"id":     id,
	})
}

func TestGroup(t *testing.T) {
	var (
		fiber      *Route
		mockConfig *configmock.Config
	)
	beforeEach := func() {
		mockConfig = &configmock.Config{}
		mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
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
				fiber.Get("/input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Json(http.StatusOK, httpcontract.Json{
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
				fiber.Post("/input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Put("/input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Delete("/input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Options("/input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Patch("/input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Any("/any/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Any("/any/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Any("/any/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Any("/any/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Any("/any/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()

				resource := resourceController{}
				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("action", "index")
					ctx.Request().Next()
				})
				fiber.Resource("/resource", resource)
			},
			method:         "GET",
			url:            "/resource",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"index\"}",
		},
		{
			name: "Resource Show",
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

				resource := resourceController{}
				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("action", "show")
					ctx.Request().Next()
				})
				fiber.Resource("/resource", resource)
			},
			method:         "GET",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"show\",\"id\":\"1\"}",
		},
		{
			name: "Resource Store",
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

				resource := resourceController{}
				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("action", "store")
					ctx.Request().Next()
				})
				fiber.Resource("/resource", resource)
			},
			method:         "POST",
			url:            "/resource",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"store\"}",
		},
		{
			name: "Resource Update (PUT)",
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

				resource := resourceController{}
				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("action", "update")
					ctx.Request().Next()
				})
				fiber.Resource("/resource", resource)
			},
			method:         "PUT",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"update\",\"id\":\"1\"}",
		},
		{
			name: "Resource Update (PATCH)",
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

				resource := resourceController{}
				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("action", "update")
					ctx.Request().Next()
				})
				fiber.Resource("/resource", resource)
			},
			method:         "PATCH",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"update\",\"id\":\"1\"}",
		},
		{
			name: "Resource Destroy",
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

				resource := resourceController{}
				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("action", "destroy")
					ctx.Request().Next()
				})
				fiber.Resource("/resource", resource)
			},
			method:         "DELETE",
			url:            "/resource/1",
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"action\":\"destroy\",\"id\":\"1\"}",
		},
		{
			name: "Static",
			setup: func(req *http.Request) {
				fiber.Static("static", "./")
			},
			method:     "GET",
			url:        "/static/README.md",
			expectCode: http.StatusOK,
		},
		{
			name: "StaticFile",
			setup: func(req *http.Request) {
				fiber.StaticFile("static-file", "./README.md")
			},
			method:     "GET",
			url:        "/static-file",
			expectCode: http.StatusOK,
		},
		{
			name: "StaticFS",
			setup: func(req *http.Request) {
				fiber.StaticFS("static-fs", http.Dir("./"))
			},
			method:     "GET",
			url:        "/static-fs/README.md",
			expectCode: http.StatusOK,
		},
		{
			name: "Abort Middleware",
			setup: func(req *http.Request) {
				fiber.Middleware(abortMiddleware()).Get("/middleware/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Middleware(contextMiddleware(), contextMiddleware1()).Get("/middlewares/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Prefix("prefix1").Prefix("prefix2").Get("input/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
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
				fiber.Prefix("group1").Middleware(contextMiddleware()).Group(func(route1 route.Router) {
					route1.Prefix("group2").Middleware(contextMiddleware1()).Group(func(route2 route.Router) {
						route2.Get("/middleware/{id}", func(ctx httpcontract.Context) {
							ctx.Response().Success().Json(httpcontract.Json{
								"id":   ctx.Request().Input("id"),
								"ctx":  ctx.Value("ctx").(string),
								"ctx1": ctx.Value("ctx1").(string),
							})
						})
					})
					route1.Middleware(contextMiddleware2()).Get("/middleware/{id}", func(ctx httpcontract.Context) {
						ctx.Response().Success().Json(httpcontract.Json{
							"id":   ctx.Request().Input("id"),
							"ctx":  ctx.Value("ctx").(string),
							"ctx2": ctx.Value("ctx2").(string),
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
				fiber.Prefix("group1").Middleware(contextMiddleware()).Group(func(route1 route.Router) {
					route1.Prefix("group2").Middleware(contextMiddleware1()).Group(func(route2 route.Router) {
						route2.Get("/middleware/{id}", func(ctx httpcontract.Context) {
							ctx.Response().Success().Json(httpcontract.Json{
								"id":   ctx.Request().Input("id"),
								"ctx":  ctx.Value("ctx").(string),
								"ctx1": ctx.Value("ctx1").(string),
							})
						})
					})
					route1.Middleware(contextMiddleware2()).Get("/middleware/{id}", func(ctx httpcontract.Context) {
						ctx.Response().Success().Json(httpcontract.Json{
							"id":   ctx.Request().Input("id"),
							"ctx":  ctx.Value("ctx").(string),
							"ctx2": ctx.Value("ctx2").(string),
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

				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.WithValue("global", "goravel")
					ctx.Request().Next()
				})
				fiber.Get("/global-middleware", func(ctx httpcontract.Context) {
					ctx.Response().Json(http.StatusOK, httpcontract.Json{
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
				fiber.Prefix("conflict").Group(func(route1 route.Router) {
					route1.Middleware(contextMiddleware()).Get("/middleware1/{id}", func(ctx httpcontract.Context) {
						ctx.Response().Success().Json(httpcontract.Json{
							"id":   ctx.Request().Input("id"),
							"ctx":  ctx.Value("ctx"),
							"ctx2": ctx.Value("ctx2"),
						})
					})
					route1.Middleware(contextMiddleware2()).Post("/middleware2/{id}", func(ctx httpcontract.Context) {
						ctx.Response().Success().Json(httpcontract.Json{
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

			fiber, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			if test.setup != nil {
				test.setup(req)
			}

			resp, err := fiber.Test(req)
			assert.NoError(t, err, test.name)

			if test.expectBodyJson != "" {
				body, _ := io.ReadAll(resp.Body)
				bodyMap := make(map[string]any)
				exceptBodyMap := make(map[string]any)

				err = sonic.Unmarshal(body, &bodyMap)
				assert.NoError(t, err, test.name)
				err = sonic.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.NoError(t, err, test.name)

				assert.Equal(t, exceptBodyMap, bodyMap, test.name)
			}

			assert.Equal(t, test.expectCode, resp.StatusCode, test.name)

			mockConfig.AssertExpectations(t)
		})
	}
}

func abortMiddleware() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		ctx.Request().AbortWithStatus(http.StatusNonAuthoritativeInfo)
	}
}

func contextMiddleware() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		ctx.WithValue("ctx", "Goravel")

		ctx.Request().Next()
	}
}

func contextMiddleware1() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		ctx.WithValue("ctx1", "Hello")

		ctx.Request().Next()
	}
}

func contextMiddleware2() httpcontract.Middleware {
	return func(ctx httpcontract.Context) {
		ctx.WithValue("ctx2", "World")

		ctx.Request().Next()
	}
}
