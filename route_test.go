package fiber

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	configmock "github.com/goravel/framework/contracts/config/mocks"
	filesystemmock "github.com/goravel/framework/contracts/filesystem/mocks"
	httpcontract "github.com/goravel/framework/contracts/http"
	logmock "github.com/goravel/framework/contracts/log/mocks"
	"github.com/goravel/framework/contracts/validation"
	validationmock "github.com/goravel/framework/contracts/validation/mocks"
	frameworkfilesystem "github.com/goravel/framework/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestFallback(t *testing.T) {
	mockConfig := &configmock.Config{}
	mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
	mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
	ConfigFacade = mockConfig

	route := NewRoute(mockConfig)
	route.Fallback(func(ctx httpcontract.Context) {
		ctx.Response().String(404, "not found")
	})

	req, err := http.NewRequest("GET", "/test", nil)
	assert.Nil(t, err)

	resp, err := route.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "not found", string(body))

	mockConfig.AssertExpectations(t)
}

func TestRun(t *testing.T) {
	var (
		mockConfig *configmock.Config
		route      *Route
	)

	tests := []struct {
		name        string
		setup       func(host string, port string) error
		host        string
		port        string
		expectError error
	}{
		{
			name: "error when default host is empty",
			setup: func(host string, port string) error {
				mockConfig.On("GetString", "http.host").Return(host).Once()

				go func() {
					assert.EqualError(t, route.Run(), "host can't be empty")
				}()
				time.Sleep(1 * time.Second)

				return errors.New("error")
			},
		},
		{
			name: "error when default port is empty",
			setup: func(host string, port string) error {
				mockConfig.On("GetString", "http.host").Return(host).Once()
				mockConfig.On("GetString", "http.port").Return(port).Once()

				go func() {
					assert.EqualError(t, route.Run(), "port can't be empty")
				}()
				time.Sleep(1 * time.Second)

				return errors.New("error")
			},
			host: "127.0.0.1",
		},
		{
			name: "use default host",
			setup: func(host string, port string) error {
				mockConfig.On("GetBool", "app.debug").Return(true).Once()
				mockConfig.On("GetString", "http.host").Return(host).Once()
				mockConfig.On("GetString", "http.port").Return(port).Once()

				go func() {
					assert.Nil(t, route.Run())
				}()

				time.Sleep(1 * time.Second)

				return nil
			},
			host: "127.0.0.1",
			port: "3031",
		},
		{
			name: "use custom host",
			setup: func(host string, port string) error {
				mockConfig.On("GetBool", "app.debug").Return(true).Once()

				go func() {
					assert.Nil(t, route.Run(host))
				}()

				return nil
			},
			host: "127.0.0.1:3032",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = &configmock.Config{}
			mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
			mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
			ConfigFacade = mockConfig

			route = NewRoute(mockConfig)
			route.Get("/", func(ctx httpcontract.Context) {
				ctx.Response().Json(200, httpcontract.Json{
					"Hello": "Goravel",
				})
			})
			if err := test.setup(test.host, test.port); err == nil {
				time.Sleep(1 * time.Second)
				hostUrl := "http://" + test.host
				if test.port != "" {
					hostUrl = hostUrl + ":" + test.port
				}
				resp, err := http.Get(hostUrl)
				assert.Nil(t, err)
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, "{\"Hello\":\"Goravel\"}", string(body))
			}

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestRunTLS(t *testing.T) {
	var (
		mockConfig *configmock.Config
		route      *Route
	)

	tests := []struct {
		name        string
		setup       func(host string, port string) error
		host        string
		port        string
		expectError error
	}{
		{
			name: "error when default host is empty",
			setup: func(host string, port string) error {
				mockConfig.On("GetString", "http.tls.host").Return(host).Once()

				go func() {
					assert.EqualError(t, route.RunTLS(), "host can't be empty")
				}()
				time.Sleep(1 * time.Second)

				return errors.New("error")
			},
		},
		{
			name: "error when default port is empty",
			setup: func(host string, port string) error {
				mockConfig.On("GetString", "http.tls.host").Return(host).Once()
				mockConfig.On("GetString", "http.tls.port").Return(port).Once()

				go func() {
					assert.EqualError(t, route.RunTLS(), "port can't be empty")
				}()
				time.Sleep(1 * time.Second)

				return errors.New("error")
			},
			host: "127.0.0.1",
		},
		{
			name: "use default host",
			setup: func(host string, port string) error {
				mockConfig.On("GetBool", "app.debug").Return(true).Once()
				mockConfig.On("GetString", "http.tls.host").Return(host).Once()
				mockConfig.On("GetString", "http.tls.port").Return(port).Once()
				mockConfig.On("GetString", "http.tls.ssl.cert").Return("test_ca.crt").Once()
				mockConfig.On("GetString", "http.tls.ssl.key").Return("test_ca.key").Once()

				go func() {
					assert.Nil(t, route.RunTLS())
				}()

				return nil
			},
			host: "127.0.0.1",
			port: "3003",
		},
		{
			name: "use custom host",
			setup: func(host string, port string) error {
				mockConfig.On("GetBool", "app.debug").Return(true).Once()
				mockConfig.On("GetString", "http.tls.ssl.cert").Return("test_ca.crt").Once()
				mockConfig.On("GetString", "http.tls.ssl.key").Return("test_ca.key").Once()

				go func() {
					assert.Nil(t, route.RunTLS(host))
				}()

				return nil
			},
			host: "127.0.0.1:3004",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = &configmock.Config{}
			mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
			mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
			ConfigFacade = mockConfig

			route = NewRoute(mockConfig)
			route.Get("/", func(ctx httpcontract.Context) {
				ctx.Response().Json(200, httpcontract.Json{
					"Hello": "Goravel",
				})
			})
			if err := test.setup(test.host, test.port); err == nil {
				time.Sleep(1 * time.Second)
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: tr}
				hostUrl := "https://" + test.host
				if test.port != "" {
					hostUrl = hostUrl + ":" + test.port
				}
				resp, err := client.Get(hostUrl)
				assert.Nil(t, err)
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, "{\"Hello\":\"Goravel\"}", string(body))
			}

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestRunTLSWithCert(t *testing.T) {
	var (
		mockConfig *configmock.Config
		route      *Route
	)

	tests := []struct {
		name        string
		setup       func(host string) error
		host        string
		expectError error
	}{
		{
			name: "error when default host is empty",
			setup: func(host string) error {
				go func() {
					assert.EqualError(t, route.RunTLSWithCert(host, "test_ca.crt", "test_ca.key"), "host can't be empty")
				}()
				time.Sleep(1 * time.Second)

				return errors.New("error")
			},
		},
		{
			name: "use default host",
			setup: func(host string) error {
				mockConfig.On("GetBool", "app.debug").Return(true).Once()

				go func() {
					assert.Nil(t, route.RunTLSWithCert(host, "test_ca.crt", "test_ca.key"))
				}()

				return nil
			},
			host: "127.0.0.1:3005",
		},
		{
			name: "use custom host",
			setup: func(host string) error {
				mockConfig.On("GetBool", "app.debug").Return(true).Once()

				go func() {
					assert.Nil(t, route.RunTLSWithCert(host, "test_ca.crt", "test_ca.key"))
				}()

				return nil
			},
			host: "127.0.0.1:3006",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = &configmock.Config{}
			mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
			mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
			ConfigFacade = mockConfig

			route = NewRoute(mockConfig)
			route.Get("/", func(ctx httpcontract.Context) {
				ctx.Response().Json(200, httpcontract.Json{
					"Hello": "Goravel",
				})
			})
			if err := test.setup(test.host); err == nil {
				time.Sleep(1 * time.Second)
				tr := &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}
				client := &http.Client{Transport: tr}
				resp, err := client.Get("https://" + test.host)
				assert.Nil(t, err)
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, "{\"Hello\":\"Goravel\"}", string(body))
			}

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestRequest(t *testing.T) {
	var (
		fiber      *Route
		req        *http.Request
		mockConfig *configmock.Config
	)
	beforeEach := func() {
		mockConfig = &configmock.Config{}
		mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
		ConfigFacade = mockConfig

		fiber = NewRoute(mockConfig)
	}
	tests := []struct {
		name           string
		method         string
		url            string
		setup          func(method, url string) error
		expectCode     int
		expectBody     string
		expectBodyJson string
	}{
		{
			name:   "All when Get and query is empty",
			method: "GET",
			url:    "/all",
			setup: func(method, url string) error {
				fiber.Get("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				req, _ = http.NewRequest(method, url, nil)

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{}}",
		},
		{
			name:   "All when Get and query is not empty",
			method: "GET",
			url:    "/all?a=1&a=2&b=3",
			setup: func(method, url string) error {
				fiber.Get("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				req, _ = http.NewRequest(method, url, nil)

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{\"a\":\"2\",\"b\":\"3\"}}",
		},
		{
			name:   "All with form when Post",
			method: "POST",
			url:    "/all?a=1&a=2&b=3",
			setup: func(method, url string) error {
				fiber.Post("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				payload := &bytes.Buffer{}
				writer := multipart.NewWriter(payload)

				if err := writer.WriteField("b", "4"); err != nil {
					return err
				}
				if err := writer.WriteField("e", "e"); err != nil {
					return err
				}

				readme, err := os.Open("./README.md")
				if err != nil {
					return err
				}
				defer readme.Close()

				part1, err := writer.CreateFormFile("file", filepath.Base("./README.md"))
				if err != nil {
					return err
				}

				if _, err = io.Copy(part1, readme); err != nil {
					return err
				}

				if err := writer.Close(); err != nil {
					return err
				}

				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return nil
			},
			expectCode: http.StatusOK,
		},
		{
			name:   "All with empty form when Post",
			method: "POST",
			url:    "/all?a=1&a=2&b=3",
			setup: func(method, url string) error {
				fiber.Post("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "multipart/form-data;boundary=0")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{\"a\":\"2\",\"b\":\"3\"}}",
		},
		{
			name:   "All with json when Post",
			method: "POST",
			url:    "/all?a=1&a=2&name=3",
			setup: func(method, url string) error {
				fiber.Post("/all", func(ctx httpcontract.Context) {
					all := ctx.Request().All()
					type Test struct {
						Name string
						Age  int
					}
					var test Test
					_ = ctx.Request().Bind(&test)

					ctx.Response().Success().Json(httpcontract.Json{
						"all":  all,
						"name": test.Name,
						"age":  test.Age,
					})
				})

				payload := strings.NewReader(`{
					"Name": "goravel",
					"Age": 1
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"age\":1,\"all\":{\"Age\":1,\"Name\":\"goravel\",\"a\":\"2\",\"name\":\"3\"},\"name\":\"goravel\"}",
		},
		{
			name:   "All with error json when Post",
			method: "POST",
			url:    "/all?a=1&a=2&name=3",
			setup: func(method, url string) error {
				mockLog := &logmock.Log{}
				LogFacade = mockLog
				mockLog.On("Error", mock.Anything).Twice()

				fiber.Post("/all", func(ctx httpcontract.Context) {
					all := ctx.Request().All()
					type Test struct {
						Name string
						Age  int
					}
					var test Test
					_ = ctx.Request().Bind(&test)

					ctx.Response().Success().Json(httpcontract.Json{
						"all":  all,
						"name": test.Name,
						"age":  test.Age,
					})
				})

				payload := strings.NewReader(`{
					"Name": "goravel",
					"Age": 1,
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"age\":1,\"all\":{\"a\":\"2\",\"name\":\"3\"},\"name\":\"goravel\"}",
		},
		{
			name:   "All with empty json when Post",
			method: "POST",
			url:    "/all?a=1&a=2&name=3",
			setup: func(method, url string) error {
				fiber.Post("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{\"a\":\"2\",\"name\":\"3\"}}",
		},
		{
			name:   "All with json when Put",
			method: "PUT",
			url:    "/all?a=1&a=2&b=3",
			setup: func(method, url string) error {
				fiber.Put("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				payload := strings.NewReader(`{
					"b": 4,
					"e": "e"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{\"a\":\"2\",\"b\":4,\"e\":\"e\"}}",
		},
		{
			name:   "All with json when Delete",
			method: "DELETE",
			url:    "/all?a=1&a=2&b=3",
			setup: func(method, url string) error {
				fiber.Delete("/all", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				payload := strings.NewReader(`{
					"b": 4,
					"e": "e"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{\"a\":\"2\",\"b\":4,\"e\":\"e\"}}",
		},
		{
			name:   "Methods",
			method: "GET",
			url:    "/methods/1?name=Goravel",
			setup: func(method, url string) error {
				fiber.Get("/methods/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id":       ctx.Request().Input("id"),
						"name":     ctx.Request().Query("name", "Hello"),
						"header":   ctx.Request().Header("Hello", "World"),
						"method":   ctx.Request().Method(),
						"path":     ctx.Request().Path(),
						"url":      ctx.Request().Url(),
						"full_url": ctx.Request().FullUrl(),
						"ip":       ctx.Request().Ip(),
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}
				req.Header.Set("Hello", "goravel")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"full_url\":\"\",\"header\":\"goravel\",\"id\":\"1\",\"ip\":\"0.0.0.0\",\"method\":\"GET\",\"name\":\"Goravel\",\"path\":\"/methods/1\",\"url\":\"/methods/1?name=Goravel\"}",
		},
		{
			name:   "Headers",
			method: "GET",
			url:    "/headers",
			setup: func(method, url string) error {
				fiber.Get("/headers", func(ctx httpcontract.Context) {
					str, _ := sonic.Marshal(ctx.Request().Headers())
					ctx.Response().Success().String(string(str))
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}
				req.Header.Set("Hello", "Goravel")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"Hello\":[\"Goravel\"],\"Content-Length\":[\"0\"]}",
		},
		{
			name:   "Route",
			method: "GET",
			url:    "/route/1/2/3/a",
			setup: func(method, url string) error {
				fiber.Get("/route/{string}/{int}/{int64}/{string1}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"string": ctx.Request().Route("string"),
						"int":    ctx.Request().RouteInt("int"),
						"int64":  ctx.Request().RouteInt64("int64"),
						"error":  ctx.Request().RouteInt("string1"),
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
			expectBodyJson: "{\"error\":0,\"int\":2,\"int64\":3,\"string\":\"1\"}",
		},
		{
			name:   "Input - from json",
			method: "POST",
			url:    "/input1/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input1/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().Input("id"),
					})
				})

				payload := strings.NewReader(`{
					"id": "3"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"3\"}",
		},
		{
			name:   "Input - from form",
			method: "POST",
			url:    "/input2/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input2/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().Input("id"),
					})
				})

				payload := &bytes.Buffer{}
				writer := multipart.NewWriter(payload)
				if err := writer.WriteField("id", "4"); err != nil {
					return err
				}
				if err := writer.Close(); err != nil {
					return err
				}

				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return nil
			},
			expectCode: http.StatusOK,
		},
		{
			name:   "Input - from json, then Bind",
			method: "POST",
			url:    "/input/json/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input/json/{id}", func(ctx httpcontract.Context) {
					id := ctx.Request().Input("id")
					var data struct {
						Name string `form:"name" json:"name"`
					}
					_ = ctx.Request().Bind(&data)
					ctx.Response().Success().Json(httpcontract.Json{
						"id":   id,
						"name": data.Name,
					})
				})

				payload := strings.NewReader(`{
					"name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"2\",\"name\":\"Goravel\"}",
		},
		{
			name:   "Input - from form, then Bind",
			method: "POST",
			url:    "/input/form/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input/form/{id}", func(ctx httpcontract.Context) {
					id := ctx.Request().Input("id")
					var data struct {
						Name string `form:"name" json:"name"`
					}
					_ = ctx.Request().Bind(&data)
					ctx.Response().Success().Json(httpcontract.Json{
						"id":   id,
						"name": data.Name,
					})
				})

				payload := &bytes.Buffer{}
				writer := multipart.NewWriter(payload)
				if err := writer.WriteField("name", "Goravel"); err != nil {
					return err
				}
				if err := writer.Close(); err != nil {
					return err
				}

				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"2\",\"name\":\"Goravel\"}",
		},
		{
			name:   "Input - from query",
			method: "POST",
			url:    "/input3/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input3/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().Input("id"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"2\"}",
		},
		{
			name:   "Input - from route",
			method: "POST",
			url:    "/input4/1",
			setup: func(method, url string) error {
				fiber.Post("/input4/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().Input("id"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"1\"}",
		},
		{
			name:   "Input - empty",
			method: "POST",
			url:    "/input5/1",
			setup: func(method, url string) error {
				fiber.Post("/input5/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id1": ctx.Request().Input("id1"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id1\":\"\"}",
		},
		{
			name:   "Input - default",
			method: "POST",
			url:    "/input6/1",
			setup: func(method, url string) error {
				fiber.Post("/input6/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id1": ctx.Request().Input("id1", "2"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id1\":\"2\"}",
		},
		{
			name:   "Input - with point",
			method: "POST",
			url:    "/input7/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input7/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().Input("id.a"),
					})
				})

				payload := strings.NewReader(`{
					"id": {"a": "3"}
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":\"3\"}",
		},
		{
			name:   "InputArray",
			method: "POST",
			url:    "/input-array/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input-array/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().InputArray("id"),
					})
				})

				payload := strings.NewReader(`{
					"id": ["3", "4"]
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":[\"3\",\"4\"]}",
		},
		{
			name:   "InputMap",
			method: "POST",
			url:    "/input-map/1?id=2",
			setup: func(method, url string) error {
				fiber.Post("/input-map/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().InputMap("id"),
					})
				})

				payload := strings.NewReader(`{
					"id": {"a": "3"}
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":{\"a\":\"3\"}}",
		},
		{
			name:   "InputInt",
			method: "POST",
			url:    "/input-int/1",
			setup: func(method, url string) error {
				fiber.Post("/input-int/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().InputInt("id"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":1}",
		},
		{
			name:   "InputInt64",
			method: "POST",
			url:    "/input-int64/1",
			setup: func(method, url string) error {
				fiber.Post("/input-int64/{id}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id": ctx.Request().InputInt64("id"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id\":1}",
		},
		{
			name:   "InputBool",
			method: "POST",
			url:    "/input-bool/1/true/on/yes/a",
			setup: func(method, url string) error {
				fiber.Post("/input-bool/{id1}/{id2}/{id3}/{id4}/{id5}", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"id1": ctx.Request().InputBool("id1"),
						"id2": ctx.Request().InputBool("id2"),
						"id3": ctx.Request().InputBool("id3"),
						"id4": ctx.Request().InputBool("id4"),
						"id5": ctx.Request().InputBool("id5"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"id1\":true,\"id2\":true,\"id3\":true,\"id4\":true,\"id5\":false}",
		},
		{
			name:   "Form",
			method: "POST",
			url:    "/form",
			setup: func(method, url string) error {
				fiber.Post("/form", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"name":  ctx.Request().Form("name", "Hello"),
						"name1": ctx.Request().Form("name1", "Hello"),
					})
				})

				payload := &bytes.Buffer{}
				writer := multipart.NewWriter(payload)
				if err := writer.WriteField("name", "Goravel"); err != nil {
					return err
				}
				if err := writer.Close(); err != nil {
					return err
				}

				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":\"Goravel\",\"name1\":\"Hello\"}",
		},
		{
			name:   "Json",
			method: "POST",
			url:    "/json",
			setup: func(method, url string) error {
				fiber.Post("/json", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"name":   ctx.Request().Json("name"),
						"info":   ctx.Request().Json("info"),
						"avatar": ctx.Request().Json("avatar", "logo"),
					})
				})

				payload := strings.NewReader(`{
					"name": "Goravel",
					"info": {"avatar": "logo"}
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"avatar\":\"logo\",\"info\":\"\",\"name\":\"Goravel\"}",
		},
		{
			name:   "Bind",
			method: "POST",
			url:    "/bind",
			setup: func(method, url string) error {
				fiber.Post("/bind", func(ctx httpcontract.Context) {
					type Test struct {
						Name string
					}
					var test Test
					_ = ctx.Request().Bind(&test)
					ctx.Response().Success().Json(httpcontract.Json{
						"name": test.Name,
					})
				})

				payload := strings.NewReader(`{
					"Name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":\"Goravel\"}",
		},
		{
			name:   "Bind, then Input",
			method: "POST",
			url:    "/bind",
			setup: func(method, url string) error {
				fiber.Post("/bind", func(ctx httpcontract.Context) {
					type Test struct {
						Name string
					}
					var test Test
					_ = ctx.Request().Bind(&test)
					ctx.Response().Success().Json(httpcontract.Json{
						"name":  test.Name,
						"name1": ctx.Request().Input("Name"),
					})
				})

				payload := strings.NewReader(`{
					"Name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":\"Goravel\",\"name1\":\"Goravel\"}",
		},
		{
			name:   "Query",
			method: "GET",
			url:    "/query?string=Goravel&int=1&int64=2&bool1=1&bool2=true&bool3=on&bool4=yes&bool5=0&error=a",
			setup: func(method, url string) error {
				fiber.Get("/query", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"string":        ctx.Request().Query("string", ""),
						"int":           ctx.Request().QueryInt("int", 11),
						"int_default":   ctx.Request().QueryInt("int_default", 11),
						"int64":         ctx.Request().QueryInt64("int64", 22),
						"int64_default": ctx.Request().QueryInt64("int64_default", 22),
						"bool1":         ctx.Request().QueryBool("bool1"),
						"bool2":         ctx.Request().QueryBool("bool2"),
						"bool3":         ctx.Request().QueryBool("bool3"),
						"bool4":         ctx.Request().QueryBool("bool4"),
						"bool5":         ctx.Request().QueryBool("bool5"),
						"error":         ctx.Request().QueryInt("error", 33),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"bool1\":true,\"bool2\":true,\"bool3\":true,\"bool4\":true,\"bool5\":false,\"error\":0,\"int\":1,\"int64\":2,\"int64_default\":22,\"int_default\":11,\"string\":\"Goravel\"}",
		},
		{
			name:   "QueryArray",
			method: "GET",
			url:    "/query-array?name=Goravel&name=Goravel1",
			setup: func(method, url string) error {
				fiber.Get("/query-array", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"name": ctx.Request().QueryArray("name"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":[\"Goravel\",\"Goravel1\"]}",
		},
		{
			name:   "QueryMap",
			method: "GET",
			url:    "/query-map?name[a]=Goravel&name[b]=Goravel1",
			setup: func(method, url string) error {
				fiber.Get("/query-map", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"name": ctx.Request().QueryMap("name"),
					})
				})

				req, _ = http.NewRequest(method, url, nil)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":{\"a\":\"Goravel\",\"b\":\"Goravel1\"}}",
		},
		{
			name:   "Queries",
			method: "GET",
			url:    "/queries?string=Goravel&int=1&int64=2&bool1=1&bool2=true&bool3=on&bool4=yes&bool5=0&error=a",
			setup: func(method, url string) error {
				fiber.Get("/queries", func(ctx httpcontract.Context) {
					ctx.Response().Success().Json(httpcontract.Json{
						"all": ctx.Request().All(),
					})
				})

				req, _ = http.NewRequest(method, url, nil)

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"all\":{\"bool1\":\"1\",\"bool2\":\"true\",\"bool3\":\"on\",\"bool4\":\"yes\",\"bool5\":\"0\",\"error\":\"a\",\"int\":\"1\",\"int64\":\"2\",\"string\":\"Goravel\"}}",
		},
		{
			name:   "File",
			method: "POST",
			url:    "/file",
			setup: func(method, url string) error {
				fiber.Post("/file", func(ctx httpcontract.Context) {
					mockConfig.On("GetString", "app.name").Return("goravel").Once()
					mockConfig.On("GetString", "filesystems.default").Return("local").Once()
					frameworkfilesystem.ConfigFacade = mockConfig

					mockStorage := &filesystemmock.Storage{}
					mockDriver := &filesystemmock.Driver{}
					mockStorage.On("Disk", "local").Return(mockDriver).Once()
					frameworkfilesystem.StorageFacade = mockStorage

					fileInfo, err := ctx.Request().File("file")

					mockDriver.On("PutFile", "test", fileInfo).Return("test/README.md", nil).Once()
					mockStorage.On("Exists", "test/README.md").Return(true).Once()

					if err != nil {
						ctx.Response().Success().String("get file error")
						return
					}
					filePath, err := fileInfo.Store("test")
					if err != nil {
						ctx.Response().Success().String("store file error: " + err.Error())
						return
					}

					extension, err := fileInfo.Extension()
					if err != nil {
						ctx.Response().Success().String("get file extension error: " + err.Error())
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"exist":              mockStorage.Exists(filePath),
						"hash_name_length":   len(fileInfo.HashName()),
						"hash_name_length1":  len(fileInfo.HashName("test")),
						"file_path_length":   len(filePath),
						"extension":          extension,
						"original_name":      fileInfo.GetClientOriginalName(),
						"original_extension": fileInfo.GetClientOriginalExtension(),
					})
				})

				payload := &bytes.Buffer{}
				writer := multipart.NewWriter(payload)
				readme, err := os.Open("./README.md")
				if err != nil {
					return err
				}
				defer readme.Close()
				part1, err := writer.CreateFormFile("file", filepath.Base("./README.md"))
				if err != nil {
					return err
				}

				if _, err = io.Copy(part1, readme); err != nil {
					return err
				}

				if err := writer.Close(); err != nil {
					return err
				}

				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", writer.FormDataContentType())

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"exist\":true,\"extension\":\"txt\",\"file_path_length\":14,\"hash_name_length\":44,\"hash_name_length1\":49,\"original_extension\":\"md\",\"original_name\":\"README.md\"}",
		},
		{
			name:   "GET with validator and validate pass",
			method: "GET",
			url:    "/validator/validate/success?name=Goravel",
			setup: func(method, url string) error {
				fiber.Get("/validator/validate/success", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					validator, err := ctx.Request().Validate(map[string]string{
						"name": "required",
					})
					if err != nil {
						ctx.Response().String(400, "Validate error: "+err.Error())
						return
					}
					if validator.Fails() {
						ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
						return
					}

					type Test struct {
						Name string `form:"name" json:"name"`
					}
					var test Test
					if err := validator.Bind(&test); err != nil {
						ctx.Response().String(400, "Validate bind error: "+err.Error())
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": test.Name,
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
			expectBodyJson: "{\"name\":\"Goravel\"}",
		},
		{
			name:   "GET with validator but validate fail",
			method: "GET",
			url:    "/validator/validate/fail?name=Goravel",
			setup: func(method, url string) error {
				fiber.Get("/validator/validate/fail", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					validator, err := ctx.Request().Validate(map[string]string{
						"name1": "required",
					})
					if err != nil {
						ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
						return
					}
					if validator.Fails() {
						ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": "",
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusBadRequest,
			expectBody: "Validate fail: map[name1:map[required:name1 is required to not be empty]]",
		},
		{
			name:   "GET with validator and validate request pass",
			method: "GET",
			url:    "/validator/validate-request/success?name=Goravel",
			setup: func(method, url string) error {
				fiber.Get("/validator/validate-request/success", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					var createUser CreateUser
					validateErrors, err := ctx.Request().ValidateRequest(&createUser)
					if err != nil {
						ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
						return
					}
					if validateErrors != nil {
						ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": createUser.Name,
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
			expectBodyJson: "{\"name\":\"Goravel1\"}",
		},
		{
			name:   "GET with validator but validate request fail",
			method: "GET",
			url:    "/validator/validate-request/fail?name1=Goravel",
			setup: func(method, url string) error {
				fiber.Get("/validator/validate-request/fail", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					var createUser CreateUser
					validateErrors, err := ctx.Request().ValidateRequest(&createUser)
					if err != nil {
						ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
						return
					}
					if validateErrors != nil {
						ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": createUser.Name,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusBadRequest,
			expectBody: "Validate fail: map[name:map[required:name is required to not be empty]]",
		},
		{
			name:   "POST with validator and validate pass",
			method: "POST",
			url:    "/validator/validate/success",
			setup: func(method, url string) error {
				fiber.Post("/validator/validate/success", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					validator, err := ctx.Request().Validate(map[string]string{
						"name": "required",
					})
					if err != nil {
						ctx.Response().String(400, "Validate error: "+err.Error())
						return
					}
					if validator.Fails() {
						ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
						return
					}

					type Test struct {
						Name string `form:"name" json:"name"`
					}
					var test Test
					if err := validator.Bind(&test); err != nil {
						ctx.Response().String(400, "Validate bind error: "+err.Error())
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": test.Name,
					})
				})

				payload := strings.NewReader(`{
					"name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":\"Goravel\"}",
		},
		{
			name:   "POST with validator and validate fail",
			method: "POST",
			url:    "/validator/validate/fail",
			setup: func(method, url string) error {
				fiber.Post("/validator/validate/fail", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					validator, err := ctx.Request().Validate(map[string]string{
						"name1": "required",
					})
					if err != nil {
						ctx.Response().String(400, "Validate error: "+err.Error())
						return
					}
					if validator.Fails() {
						ctx.Response().String(400, fmt.Sprintf("Validate fail: %+v", validator.Errors().All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": "",
					})
				})
				payload := strings.NewReader(`{
					"name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode: http.StatusBadRequest,
			expectBody: "Validate fail: map[name1:map[required:name1 is required to not be empty]]",
		},
		{
			name:   "POST with validator and validate request pass",
			method: "POST",
			url:    "/validator/validate-request/success",
			setup: func(method, url string) error {
				fiber.Post("/validator/validate-request/success", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					var createUser CreateUser
					validateErrors, err := ctx.Request().ValidateRequest(&createUser)
					if err != nil {
						ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
						return
					}
					if validateErrors != nil {
						ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": createUser.Name,
					})
				})

				payload := strings.NewReader(`{
					"name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode:     http.StatusOK,
			expectBodyJson: "{\"name\":\"Goravel1\"}",
		},
		{
			name:   "POST with validator and validate request fail",
			method: "POST",
			url:    "/validator/validate-request/fail",
			setup: func(method, url string) error {
				fiber.Post("/validator/validate-request/fail", func(ctx httpcontract.Context) {
					mockValication := &validationmock.Validation{}
					mockValication.On("Rules").Return([]validation.Rule{}).Once()
					ValidationFacade = mockValication

					var createUser CreateUser
					validateErrors, err := ctx.Request().ValidateRequest(&createUser)
					if err != nil {
						ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
						return
					}
					if validateErrors != nil {
						ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": createUser.Name,
					})
				})

				payload := strings.NewReader(`{
					"name1": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode: http.StatusBadRequest,
			expectBody: "Validate fail: map[name:map[required:name is required to not be empty]]",
		},
		{
			name:   "POST with validator and validate request unauthorize",
			method: "POST",
			url:    "/validator/validate-request/unauthorize",
			setup: func(method, url string) error {
				fiber.Post("/validator/validate-request/unauthorize", func(ctx httpcontract.Context) {
					var unauthorize Unauthorize
					validateErrors, err := ctx.Request().ValidateRequest(&unauthorize)
					if err != nil {
						ctx.Response().String(http.StatusBadRequest, "Validate error: "+err.Error())
						return
					}
					if validateErrors != nil {
						ctx.Response().String(http.StatusBadRequest, fmt.Sprintf("Validate fail: %+v", validateErrors.All()))
						return
					}

					ctx.Response().Success().Json(httpcontract.Json{
						"name": unauthorize.Name,
					})
				})
				payload := strings.NewReader(`{
					"name": "Goravel"
				}`)
				req, _ = http.NewRequest(method, url, payload)
				req.Header.Set("Content-Type", "application/json")

				return nil
			},
			expectCode: http.StatusBadRequest,
			expectBody: "Validate error: error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			beforeEach()
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

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestResponse(t *testing.T) {
	var (
		fiber      *Route
		req        *http.Request
		mockConfig *configmock.Config
	)
	beforeEach := func() {
		mockConfig = &configmock.Config{}
		mockConfig.On("GetBool", "app.debug", false).Return(true).Once()
		mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
		ConfigFacade = mockConfig

		fiber = NewRoute(mockConfig)
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
			err := test.setup(test.method, test.url)
			assert.Nil(t, err)

			resp, err := fiber.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, _ := io.ReadAll(resp.Body)
				assert.Equal(t, test.expectBody, string(body))
			}
			if test.expectBodyJson != "" {
				body, _ := io.ReadAll(resp.Body)
				bodyMap := make(map[string]any)
				exceptBodyMap := make(map[string]any)

				err = sonic.Unmarshal(body, &bodyMap)
				assert.NoError(t, err)
				err = sonic.UnmarshalString(test.expectBodyJson, &exceptBodyMap)
				assert.NoError(t, err)

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

type CreateUser struct {
	Name string `form:"name" json:"name"`
}

func (r *CreateUser) Authorize(ctx httpcontract.Context) error {
	return nil
}

func (r *CreateUser) Rules(ctx httpcontract.Context) map[string]string {
	return map[string]string{
		"name": "required",
	}
}

func (r *CreateUser) Messages(ctx httpcontract.Context) map[string]string {
	return map[string]string{}
}

func (r *CreateUser) Attributes(ctx httpcontract.Context) map[string]string {
	return map[string]string{}
}

func (r *CreateUser) PrepareForValidation(ctx httpcontract.Context, data validation.Data) error {
	if name, exist := data.Get("name"); exist {
		return data.Set("name", name.(string)+"1")
	}

	return nil
}

type Unauthorize struct {
	Name string `form:"name" json:"name"`
}

func (r *Unauthorize) Authorize(ctx httpcontract.Context) error {
	return errors.New("error")
}

func (r *Unauthorize) Rules(ctx httpcontract.Context) map[string]string {
	return map[string]string{
		"name": "required",
	}
}

func (r *Unauthorize) Messages(ctx httpcontract.Context) map[string]string {
	return map[string]string{}
}

func (r *Unauthorize) Attributes(ctx httpcontract.Context) map[string]string {
	return map[string]string{}
}

func (r *Unauthorize) PrepareForValidation(ctx httpcontract.Context, data validation.Data) error {
	return nil
}
