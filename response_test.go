package fiber

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	configmock "github.com/goravel/framework/contracts/config/mocks"
	httpcontract "github.com/goravel/framework/contracts/http"
	"github.com/stretchr/testify/assert"
)

func TestResponse(t *testing.T) {
	var (
		err        error
		fiber      *Route
		req        *http.Request
		mockConfig *configmock.Config
	)
	beforeEach := func() {
		mockConfig = &configmock.Config{}
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
		ConfigFacade = mockConfig
	}
	tests := []struct {
		name           string
		method         string
		url            string
		setup          func(method, url string) error
		expectCode     int
		expectBody     string
		expectBodyJson string
		expectHeader   string
	}{
		{
			name:   "Data",
			method: "GET",
			url:    "/data",
			setup: func(method, url string) error {
				fiber.Get("/data", func(ctx httpcontract.Context) {
					ctx.Response().Data(http.StatusOK, "text/html; charset=utf-8", []byte("<b>Goravel</b>"))
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "<b>Goravel</b>",
		},
		{
			name:   "Success Data",
			method: "GET",
			url:    "/success/data",
			setup: func(method, url string) error {
				fiber.Get("/success/data", func(ctx httpcontract.Context) {
					ctx.Response().Success().Data("text/html; charset=utf-8", []byte("<b>Goravel</b>"))
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "<b>Goravel</b>",
		},
		{
			name:   "Json",
			method: "GET",
			url:    "/json",
			setup: func(method, url string) error {
				fiber.Get("/json", func(ctx httpcontract.Context) {
					ctx.Response().Json(http.StatusOK, httpcontract.Json{
						"id": "1",
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name:   "String",
			method: "GET",
			url:    "/string",
			setup: func(method, url string) error {
				fiber.Get("/string", func(ctx httpcontract.Context) {
					ctx.Response().String(http.StatusCreated, "Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusCreated,
			expectBody: "Goravel",
		},
		{
			name:   "Success Json",
			method: "GET",
			url:    "/success/json",
			setup: func(method, url string) error {
				fiber.Get("/success/json", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": "1",
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name:   "Success String",
			method: "GET",
			url:    "/success/string",
			setup: func(method, url string) error {
				fiber.Get("/success/string", func(ctx httpcontract.Context) {
					ctx.Response().Success().String("Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "Goravel",
		},
		{
			name:   "File",
			method: "GET",
			url:    "/file",
			setup: func(method, url string) error {
				fiber.Get("/file", func(ctx httpcontract.Context) {
					ctx.Response().File("./README.md")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
		},
		{
			name:   "Download",
			method: "GET",
			url:    "/download",
			setup: func(method, url string) error {
				fiber.Get("/download", func(ctx httpcontract.Context) {
					ctx.Response().Download("./README.md", "README.md")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
		},
		{
			name:   "Header",
			method: "GET",
			url:    "/header",
			setup: func(method, url string) error {
				fiber.Get("/header", func(ctx httpcontract.Context) {
					ctx.Response().Header("Hello", "goravel").String(http.StatusOK, "Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode:   http.StatusOK,
			expectBody:   "Goravel",
			expectHeader: "goravel",
		},
		{
			name:   "Origin",
			method: "GET",
			url:    "/origin",
			setup: func(method, url string) error {
				mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
				mockConfig.On("GetString", "app.timezone", "UTC").Return("UTC").Once()
				mockConfig.On("Get", "cors.paths").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_methods").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_origins").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.allowed_headers").Return([]string{"*"}).Once()
				mockConfig.On("Get", "cors.exposed_headers").Return([]string{"*"}).Once()
				mockConfig.On("GetInt", "cors.max_age").Return(0).Once()
				mockConfig.On("GetBool", "cors.supports_credentials").Return(false).Once()
				ConfigFacade = mockConfig

				fiber.GlobalMiddleware(func(ctx httpcontract.Context) {
					ctx.Response().Header("global", "goravel")
					ctx.Request().Next()

					assert.Equal(t, "Goravel", ctx.Response().Origin().Body().String())
					assert.Equal(t, "goravel", ctx.Response().Origin().Header().Get("global"))
					assert.Equal(t, 7, ctx.Response().Origin().Size())
					assert.Equal(t, 200, ctx.Response().Origin().Status())
				})
				fiber.Get("/origin", func(ctx httpcontract.Context) {
					ctx.Response().String(http.StatusOK, "Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "Goravel",
		},
		{
			name:   "Redirect",
			method: "GET",
			url:    "/redirect",
			setup: func(method, url string) error {
				fiber.Get("/redirect", func(ctx httpcontract.Context) {
					ctx.Response().Redirect(http.StatusMovedPermanently, "https://goravel.dev")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusMovedPermanently,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			fiber, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := fiber.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}
			if test.expectBodyJson != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				bodyMap := make(map[string]any)
				exceptBodyMap := make(map[string]any)

				err = sonic.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)
				err = sonic.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.Nil(t, err)

				assert.Equal(t, exceptBodyMap, bodyMap)
			}
			if test.expectHeader != "" {
				assert.Equal(t, test.expectHeader, strings.Join(resp.Header.Values("Hello"), ""))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestResponse_Success(t *testing.T) {
	var (
		err        error
		fiber      *Route
		req        *http.Request
		mockConfig *configmock.Config
	)
	beforeEach := func() {
		mockConfig = &configmock.Config{}
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
		ConfigFacade = mockConfig
	}
	tests := []struct {
		name           string
		method         string
		url            string
		setup          func(method, url string) error
		expectCode     int
		expectBody     string
		expectBodyJson string
		expectHeader   string
	}{
		{
			name:   "Data",
			method: "GET",
			url:    "/data",
			setup: func(method, url string) error {
				fiber.Get("/data", func(ctx httpcontract.Context) {
					ctx.Response().Success().Data("text/html; charset=utf-8", []byte("<b>Goravel</b>"))
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "<b>Goravel</b>",
		},
		{
			name:   "Json",
			method: "GET",
			url:    "/json",
			setup: func(method, url string) error {
				fiber.Get("/json", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": "1",
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name:   "String",
			method: "GET",
			url:    "/string",
			setup: func(method, url string) error {
				fiber.Get("/string", func(ctx httpcontract.Context) {
					ctx.Response().Success().String("Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "Goravel",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			fiber, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := fiber.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}
			if test.expectBodyJson != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				bodyMap := make(map[string]any)
				exceptBodyMap := make(map[string]any)

				err = sonic.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)
				err = sonic.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.Nil(t, err)

				assert.Equal(t, exceptBodyMap, bodyMap)
			}
			if test.expectHeader != "" {
				assert.Equal(t, test.expectHeader, strings.Join(resp.Header.Values("Hello"), ""))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestResponse_Status(t *testing.T) {
	var (
		err        error
		fiber      *Route
		req        *http.Request
		mockConfig *configmock.Config
	)
	beforeEach := func() {
		mockConfig = &configmock.Config{}
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
		ConfigFacade = mockConfig
	}
	tests := []struct {
		name           string
		method         string
		url            string
		setup          func(method, url string) error
		expectCode     int
		expectBody     string
		expectBodyJson string
		expectHeader   string
	}{
		{
			name:   "Data",
			method: "GET",
			url:    "/data",
			setup: func(method, url string) error {
				fiber.Get("/data", func(ctx httpcontract.Context) {
					ctx.Response().Status(http.StatusCreated).Data("text/html; charset=utf-8", []byte("<b>Goravel</b>"))
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusCreated,
			expectBody: "<b>Goravel</b>",
		},
		{
			name:   "Json",
			method: "GET",
			url:    "/json",
			setup: func(method, url string) error {
				fiber.Get("/json", func(ctx httpcontract.Context) {
					ctx.Response().Status(http.StatusCreated).Json(httpcontract.Json{
						"id": "1",
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode:     http.StatusCreated,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name:   "String",
			method: "GET",
			url:    "/string",
			setup: func(method, url string) error {
				fiber.Get("/string", func(ctx httpcontract.Context) {
					ctx.Response().Status(http.StatusCreated).String("Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusCreated,
			expectBody: "Goravel",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			fiber, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := fiber.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}
			if test.expectBodyJson != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				bodyMap := make(map[string]any)
				exceptBodyMap := make(map[string]any)

				err = sonic.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)
				err = sonic.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.Nil(t, err)

				assert.Equal(t, exceptBodyMap, bodyMap)
			}
			if test.expectHeader != "" {
				assert.Equal(t, test.expectHeader, strings.Join(resp.Header.Values("Hello"), ""))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
		})
	}
}