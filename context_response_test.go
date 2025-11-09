package fiber

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	contractshttp "github.com/goravel/framework/contracts/http"
	mocksconfig "github.com/goravel/framework/mocks/config"
	"github.com/goravel/framework/support/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse(t *testing.T) {
	var (
		err        error
		route      *Route
		req        *http.Request
		mockConfig *mocksconfig.Config
	)
	beforeEach := func() {
		mockConfig = mocksconfig.NewConfig(t)
		mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
		mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
		mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
		mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()
		ConfigFacade = mockConfig
	}
	tests := []struct {
		name                string
		method              string
		url                 string
		cookieName          string
		setup               func(method, url string) error
		expectCode          int
		expectBody          string
		expectBodyJson      string
		expectHeader        string
		expectedCookieValue string
	}{
		{
			name:   "Data",
			method: "GET",
			url:    "/data",
			setup: func(method, url string) error {
				route.Get("/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Data(http.StatusOK, "text/html; charset=utf-8", []byte("<b>Goravel</b>"))
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
				route.Get("/success/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Data("text/html; charset=utf-8", []byte("<b>Goravel</b>"))
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
				route.Get("/json", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Json(http.StatusOK, contractshttp.Json{
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
				route.Get("/string", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().String(http.StatusCreated, "Goravel")
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
				route.Get("/success/json", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
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
				route.Get("/success/string", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().String("Goravel")
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
				route.Get("/file", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().File("./README.md")
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
				route.Get("/download", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Download("./README.md", "README.md")
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
				route.Get("/header", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Header("Hello", "goravel").String(http.StatusOK, "Goravel")
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
			name:   "NoContent",
			method: "GET",
			url:    "/no/content",
			setup: func(method, url string) error {
				route.Get("/no/content", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().NoContent()
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}
				return nil
			},
			expectCode: http.StatusNoContent,
		},
		{
			name:   "NoContentWithCode",
			method: "GET",
			url:    "/no/content/with/code",
			setup: func(method, url string) error {
				route.Get("/no/content/with/code", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().NoContent(http.StatusCreated)
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}
				return nil
			},
			expectCode: http.StatusCreated,
		},
		{
			name:   "Origin",
			method: "GET",
			url:    "/origin",
			setup: func(method, url string) error {
				route.setMiddlewares([]fiber.Handler{
					middlewareToFiberHandler(func(ctx contractshttp.Context) {
						ctx.Response().Header("global", "goravel")
						ctx.Request().Next()

						assert.Equal(t, "Goravel", ctx.Response().Origin().Body().String())
						assert.Equal(t, "goravel", ctx.Response().Origin().Header().Get("global"))
						assert.Equal(t, 7, ctx.Response().Origin().Size())
						assert.Equal(t, 200, ctx.Response().Origin().Status())
					}),
				})
				route.Get("/origin", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().String(http.StatusOK, "Goravel")
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
				route.Get("/redirect", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Redirect(http.StatusMovedPermanently, "https://goravel.dev")
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
		{
			name:   "Writer",
			method: "GET",
			url:    "/writer",
			setup: func(method, url string) error {
				route.Get("/writer", func(ctx contractshttp.Context) contractshttp.Response {
					_, err = fmt.Fprintf(ctx.Response().Writer(), "Goravel")
					return nil
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
			name:       "WithoutCookie",
			method:     "GET",
			url:        "/without/cookie",
			cookieName: "goravel",
			setup: func(method, url string) error {
				route.Get("/without/cookie", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().WithoutCookie("goravel").String(http.StatusOK, "Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}
				req.AddCookie(&http.Cookie{
					Name:  "goravel",
					Value: "goravel",
				})

				return nil
			},
			expectCode:          http.StatusOK,
			expectBody:          "Goravel",
			expectedCookieValue: "",
		},
		{
			name:       "Cookie",
			method:     "GET",
			url:        "/cookie",
			cookieName: "goravel",
			setup: func(method, url string) error {
				route.Get("/cookie", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Cookie(contractshttp.Cookie{
						Name:  "goravel",
						Value: "goravel",
					}).String(http.StatusOK, "Goravel")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode:          http.StatusOK,
			expectBody:          "Goravel",
			expectedCookieValue: "goravel",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
			route = &Route{
				config: mockConfig,
				driver: "fiber",
			}
			err = route.init(nil)
			require.Nil(t, err)

			err = test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := route.Test(req)
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

				err = json.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)
				err = json.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.Nil(t, err)

				assert.Equal(t, exceptBodyMap, bodyMap)
			}
			if test.expectHeader != "" {
				assert.Equal(t, test.expectHeader, strings.Join(resp.Header.Values("Hello"), ""))
			}

			if test.cookieName != "" {
				cookies := resp.Cookies()
				exist := false
				for _, cookie := range cookies {
					if cookie.Name == test.cookieName {
						exist = true
						assert.Equal(t, test.expectedCookieValue, cookie.Value)
					}
				}
				assert.True(t, exist)
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)
		})
	}
}

func TestResponse_Success(t *testing.T) {
	var (
		err        error
		route      *Route
		req        *http.Request
		mockConfig *mocksconfig.Config
	)
	beforeEach := func() {
		mockConfig = mocksconfig.NewConfig(t)
		mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
		mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
		mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
		mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()
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
				route.Get("/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Data("text/html; charset=utf-8", []byte("<b>Goravel</b>"))
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
				route.Get("/json", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().Json(contractshttp.Json{
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
				route.Get("/string", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Success().String("Goravel")
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
			route = &Route{
				config: mockConfig,
				driver: "fiber",
			}
			err = route.init(nil)
			require.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := route.Test(req)
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

				err = json.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)
				err = json.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.Nil(t, err)

				assert.Equal(t, exceptBodyMap, bodyMap)
			}
			if test.expectHeader != "" {
				assert.Equal(t, test.expectHeader, strings.Join(resp.Header.Values("Hello"), ""))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)
		})
	}
}

func TestResponse_Status(t *testing.T) {
	var (
		err        error
		route      *Route
		req        *http.Request
		mockConfig *mocksconfig.Config
	)
	beforeEach := func() {
		mockConfig = mocksconfig.NewConfig(t)
		mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
		mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
		mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
		mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
		mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
		mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()
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
				route.Get("/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Status(http.StatusCreated).Data("text/html; charset=utf-8", []byte("<b>Goravel</b>"))
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
				route.Get("/json", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Status(http.StatusCreated).Json(contractshttp.Json{
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
				route.Get("/string", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Status(http.StatusCreated).String("Goravel")
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
			route = &Route{
				config: mockConfig,
				driver: "fiber",
			}
			err = route.init(nil)
			require.Nil(t, err)

			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := route.Test(req)
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

				err = json.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)
				err = json.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.Nil(t, err)

				assert.Equal(t, exceptBodyMap, bodyMap)
			}
			if test.expectHeader != "" {
				assert.Equal(t, test.expectHeader, strings.Join(resp.Header.Values("Hello"), ""))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)
		})
	}
}

func TestResponse_Stream(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
	mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()

	route := &Route{
		config: mockConfig,
		driver: "fiber",
	}
	err := route.init(nil)
	require.Nil(t, err)

	route.Get("/stream", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Status(http.StatusCreated).Stream(func(w contractshttp.StreamWriter) error {
			b := []string{"a", "b", "c"}
			for _, a := range b {
				if _, err := w.Write([]byte(a + "\n")); err != nil {
					return err
				}

				if err := w.Flush(); err != nil {
					return err
				}
			}

			return nil
		})
	})

	req, err := http.NewRequest("GET", "/stream", nil)
	assert.Nil(t, err)

	resp, err := route.Test(req)
	assert.NoError(t, err)

	scanner := bufio.NewScanner(resp.Body)
	var output []string
	for scanner.Scan() {
		output = append(output, scanner.Text())
	}

	assert.Equal(t, []string{"a", "b", "c"}, output)

}
