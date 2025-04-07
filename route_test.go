package fiber

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	contractshttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
	mocksconfig "github.com/goravel/framework/mocks/config"
	"github.com/goravel/framework/support/path"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
)

func TestRecoverWithCustomCallback(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)

	mockConfig.On("GetBool", "http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.On("GetBool", "http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.On("GetInt", "http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.On("GetInt", "http.drivers.fiber.header_limit", 4096).Return(4096).Once()

	route, err := NewRoute(mockConfig, nil)
	assert.Nil(t, err)

	globalRecoverCallback := func(ctx contractshttp.Context, err any) {
		ctx.Request().Abort(http.StatusInternalServerError)
	}

	route.Recover(globalRecoverCallback)

	route.Get("/recover", func(ctx contractshttp.Context) contractshttp.Response {
		panic(1)
	})

	req := httptest.NewRequest("GET", "/recover", nil)
	resp, err := route.Test(req)
	assert.Nil(t, err)

	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "Internal Server Error", string(body))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestFallback(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	ConfigFacade = mockConfig

	route, err := NewRoute(mockConfig, nil)
	assert.Nil(t, err)

	route.Fallback(func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusNotFound, "not found")
	})
	route.Get("/test", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, "test")
	})

	req, err := http.NewRequest("GET", "/test", nil)
	assert.Nil(t, err)
	resp, err := route.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "test", string(body))

	req, err = http.NewRequest("GET", "/not-found", nil)
	assert.Nil(t, err)

	resp, err = route.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "not found", string(body))
}

func TestGlobalMiddleware(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().GetBool("app.debug", false).Return(false).Twice()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Twice()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Twice()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Twice()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Twice()

	// has timeout middleware
	mockConfig.EXPECT().GetInt("http.request_timeout", 3).Return(1).Once()
	route, err := NewRoute(mockConfig, nil)
	assert.Nil(t, err)
	route.GlobalMiddleware()
	assert.Equal(t, route.instance.HandlersCount(), uint32(3))

	// no timeout middleware
	mockConfig.EXPECT().GetInt("http.request_timeout", 3).Return(0).Once()
	route, err = NewRoute(mockConfig, nil)
	assert.Nil(t, err)
	route.GlobalMiddleware()
	assert.Equal(t, route.instance.HandlersCount(), uint32(2))

}

func TestListen(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
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
			name: "success listen",
			setup: func(host string, port string) error {
				go func() {
					l, err := net.Listen("tcp", host)
					assert.Nil(t, err)
					assert.Nil(t, route.Listen(l))
				}()
				time.Sleep(1 * time.Second)

				return errors.New("error")
			},
			host: "127.0.0.1:3100",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)
			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Json(200, contractshttp.Json{
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
		})
	}
}

func TestListenTLS(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
		route      *Route
	)

	tests := []struct {
		name        string
		setup       func(host string) error
		host        string
		expectError error
	}{
		{
			name: "success listen",
			setup: func(host string) error {
				go func() {
					l, err := net.Listen("tcp", host)
					assert.Nil(t, err)
					assert.Nil(t, route.ListenTLS(l))
				}()

				return nil
			},
			host: "127.0.0.1:3101",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetString("http.tls.ssl.cert").Return("test_ca.crt").Once()
			mockConfig.EXPECT().GetString("http.tls.ssl.key").Return("test_ca.key").Once()
			ConfigFacade = mockConfig

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)
			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Json(200, contractshttp.Json{
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
		})
	}
}

func TestListenTLSWithCert(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
		route      *Route
	)

	tests := []struct {
		name        string
		setup       func(host string) error
		host        string
		expectError error
	}{
		{
			name: "success listen",
			setup: func(host string) error {
				go func() {
					l, err := net.Listen("tcp", host)
					assert.Nil(t, err)
					assert.Nil(t, route.ListenTLSWithCert(l, "test_ca.crt", "test_ca.key"))
				}()

				return nil
			},
			host: "127.0.0.1:3102",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			ConfigFacade = mockConfig

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)
			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Json(200, contractshttp.Json{
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
		})
	}
}

func TestRoutesNoRoutes(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()

	route, err := NewRoute(mockConfig, nil)
	assert.Nil(t, err)

	routes := route.Routes()
	assert.Empty(t, routes)
}

func TestRoutesWithMultipleRoutes(t *testing.T) {
	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()

	route, err := NewRoute(mockConfig, nil)
	assert.Nil(t, err)

	route.Get("/users", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(200, "List Users")
	})
	route.Post("/users", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(201, "Create User")
	})
	route.Put("/users/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(200, "Update User")
	})
	route.Delete("/users/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(204, "Delete User")
	})

	routes := route.Routes()
	assert.Len(t, routes, 5)

	routeFound := make(map[string]bool)
	for _, r := range routes {
		routeFound[r.Method+":"+r.Path] = true
	}

	assert.True(t, routeFound["HEAD:/users"])
	assert.True(t, routeFound["GET:/users"])
	assert.True(t, routeFound["POST:/users"])
	assert.True(t, routeFound["PUT:/users/{id}"])
	assert.True(t, routeFound["DELETE:/users/{id}"])
}

func TestRun(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
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
			name: "error when default port is empty",
			setup: func(host string, port string) error {
				mockConfig.EXPECT().GetString("http.host").Return(host).Once()
				mockConfig.EXPECT().GetString("http.port").Return(port).Once()

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
				mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
				mockConfig.EXPECT().GetString("http.host").Return(host).Once()
				mockConfig.EXPECT().GetString("http.port").Return(port).Once()

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
				mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

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
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			ConfigFacade = mockConfig

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Json(200, contractshttp.Json{
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
		})
	}
}

func TestRunTLS(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
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
			name: "error when default port is empty",
			setup: func(host string, port string) error {
				mockConfig.EXPECT().GetString("http.tls.host").Return(host).Once()
				mockConfig.EXPECT().GetString("http.tls.port").Return(port).Once()

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
				mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
				mockConfig.EXPECT().GetString("http.tls.host").Return(host).Once()
				mockConfig.EXPECT().GetString("http.tls.port").Return(port).Once()
				mockConfig.EXPECT().GetString("http.tls.ssl.cert").Return("test_ca.crt").Once()
				mockConfig.EXPECT().GetString("http.tls.ssl.key").Return("test_ca.key").Once()

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
				mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
				mockConfig.EXPECT().GetString("http.tls.ssl.cert").Return("test_ca.crt").Once()
				mockConfig.EXPECT().GetString("http.tls.ssl.key").Return("test_ca.key").Once()

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
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			ConfigFacade = mockConfig

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Json(200, contractshttp.Json{
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
		})
	}
}

func TestRunTLSWithCert(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
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
				mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

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
				mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

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
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			ConfigFacade = mockConfig

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				return ctx.Response().Json(200, contractshttp.Json{
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

		})
	}
}

func TestNewRoute(t *testing.T) {
	var mockConfig *mocksconfig.Config
	template := html.New(path.Resource("views"), ".tmpl")

	tests := []struct {
		name           string
		parameters     map[string]any
		setup          func()
		expectTemplate fiber.Views
		expectError    error
	}{
		{
			name: "parameters is nil",
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			},
			expectTemplate: nil,
		},
		{
			name:       "template is instance",
			parameters: map[string]any{"driver": "fiber"},
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(template).Once()
			},
			expectTemplate: template,
		},
		{
			name:       "template is callback and returns success",
			parameters: map[string]any{"driver": "fiber"},
			setup: func() {
				mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
				mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
				mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(func() (fiber.Views, error) {
					return template, nil
				}).Twice()
			},
			expectTemplate: template,
		},
		{
			name:       "template is callback and returns error",
			parameters: map[string]any{"driver": "fiber"},
			setup: func() {
				mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(func() (fiber.Views, error) {
					return nil, errors.New("error")
				}).Twice()
			},
			expectError: errors.New("error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = mocksconfig.NewConfig(t)
			test.setup()
			route, err := NewRoute(mockConfig, test.parameters)
			assert.Equal(t, test.expectError, err)
			if route != nil {
				assert.IsType(t, test.expectTemplate, route.instance.Config().Views)
			}

		})
	}
}

func TestShutdown(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
		route      *Route
		count      atomic.Int64
		host       = "127.0.0.1"
		port       = "6789"
		addr       = fmt.Sprintf("http://%s:%s", host, port)
	)

	tests := []struct {
		name  string
		setup func() error
	}{
		{
			name: "no new requests will be accepted after shutdown",
			setup: func() error {
				go func() {
					assert.Nil(t, route.Run())
				}()

				time.Sleep(1 * time.Second)

				assertHttpNormal(t, addr, true)

				assert.Nil(t, route.Shutdown())

				assertHttpNormal(t, addr, false)
				return nil
			},
		},
		{
			name: "Ensure that received requests are processed",
			setup: func() error {
				go func() {
					assert.Nil(t, route.Run())
				}()

				time.Sleep(1 * time.Second)

				wg := sync.WaitGroup{}
				count.Store(0)
				for i := 0; i < 3; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						assertHttpNormal(t, addr, true)
					}()
				}
				time.Sleep(100 * time.Millisecond)
				assert.Nil(t, route.Shutdown())
				assertHttpNormal(t, addr, false)
				wg.Wait()
				assert.Equal(t, count.Load(), int64(3))
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("app.debug").Return(true)
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetString("http.host").Return(host).Once()
			mockConfig.EXPECT().GetString("http.port").Return(port).Once()
			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)
			route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
				time.Sleep(time.Second)
				defer count.Add(1)
				return ctx.Response().Success().String("Goravel")
			})
			if err := test.setup(); err == nil {
				assert.Nil(t, err)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	var (
		err        error
		mockConfig *mocksconfig.Config
		route      *Route
	)

	tests := []struct {
		name         string
		setup        func() error
		method       string
		path         string
		body         string
		contentType  string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "serve_HTTP_GET_request",
			method:       "GET",
			path:         "/test-serve-http",
			expectedCode: http.StatusOK,
			expectedBody: `{"message":"Hello from GET"}`,
			setup: func() error {
				route.Get("/test-serve-http", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().Json(http.StatusOK, contractshttp.Json{
						"message": "Hello from GET",
					})
				})
				return nil
			},
		},
		{
			name:         "serve_HTTP_POST_request",
			method:       "POST",
			path:         "/api/users",
			body:         `{"name":"John Doe"}`,
			contentType:  "application/json",
			expectedCode: http.StatusOK,
			expectedBody: `{"name":"John Doe","status":"success"}`,
			setup: func() error {
				route.Post("/api/users", func(ctx contractshttp.Context) contractshttp.Response {
					name := ctx.Request().Input("name", "")
					return ctx.Response().Json(http.StatusOK, contractshttp.Json{
						"status": "success",
						"name":   name,
					})
				})
				return nil
			},
		},
		{
			name:         "Serve Not Found request",
			method:       "GET",
			path:         "/not-found",
			expectedCode: http.StatusNotFound,
			setup:        func() error { return nil },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockConfig = mocksconfig.NewConfig(t)
			mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
			mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
			mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()

			route, err = NewRoute(mockConfig, nil)
			assert.Nil(t, err)

			err = test.setup()
			assert.Nil(t, err)

			var req *http.Request
			if test.body != "" {
				req = httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			} else {
				req = httptest.NewRequest(test.method, test.path, nil)
			}

			if test.contentType != "" {
				req.Header.Set("Content-Type", test.contentType)
			}

			rec := httptest.NewRecorder()
			route.ServeHTTP(rec, req)

			resp := rec.Result()
			assert.Equal(t, test.expectedCode, resp.StatusCode)

			if test.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.JSONEq(t, test.expectedBody, string(body))
			}
		})
	}
}

func assertHttpNormal(t *testing.T, addr string, expectNormal bool) {
	resp, err := http.DefaultClient.Get(addr)
	if !expectNormal {
		assert.NotNil(t, err)
		assert.Nil(t, resp)
	} else {
		assert.Nil(t, err)
		assert.NotNil(t, resp)
		if resp != nil {
			assert.Equal(t, resp.StatusCode, http.StatusOK)
			body, err := io.ReadAll(resp.Body)
			assert.Nil(t, err)
			assert.Equal(t, string(body), "Goravel")
		}
	}
}

type CreateUser struct {
	Name string `form:"name" json:"name"`
}

func (r *CreateUser) Authorize(ctx contractshttp.Context) error {
	return nil
}

func (r *CreateUser) Filters(ctx contractshttp.Context) map[string]string {
	return map[string]string{
		"name": "trim",
	}
}

func (r *CreateUser) Rules(ctx contractshttp.Context) map[string]string {
	return map[string]string{
		"name": "required",
	}
}

func (r *CreateUser) Messages(ctx contractshttp.Context) map[string]string {
	return map[string]string{}
}

func (r *CreateUser) Attributes(ctx contractshttp.Context) map[string]string {
	return map[string]string{}
}

func (r *CreateUser) PrepareForValidation(ctx contractshttp.Context, data validation.Data) error {
	test := cast.ToString(ctx.Value("test"))
	if name, exist := data.Get("name"); exist {
		return data.Set("name", name.(string)+"1"+test)
	}

	return nil
}

type Unauthorize struct {
	Name string `form:"name" json:"name"`
}

func (r *Unauthorize) Authorize(ctx contractshttp.Context) error {
	return errors.New("error")
}

func (r *Unauthorize) Rules(ctx contractshttp.Context) map[string]string {
	return map[string]string{
		"name": "required",
	}
}

func (r *Unauthorize) Filters(ctx contractshttp.Context) map[string]string {
	return map[string]string{
		"name": "trim",
	}
}

func (r *Unauthorize) Messages(ctx contractshttp.Context) map[string]string {
	return map[string]string{}
}

func (r *Unauthorize) Attributes(ctx contractshttp.Context) map[string]string {
	return map[string]string{}
}

func (r *Unauthorize) PrepareForValidation(ctx contractshttp.Context, data validation.Data) error {
	return nil
}

type FileImageJson struct {
	Name  string                `form:"name" json:"name"`
	File  *multipart.FileHeader `form:"file" json:"file"`
	Image *multipart.FileHeader `form:"image" json:"image"`
	Json  string                `form:"json" json:"json"`
}

func (r *FileImageJson) Authorize(ctx contractshttp.Context) error {
	return nil
}

func (r *FileImageJson) Filters(ctx contractshttp.Context) map[string]string {
	return map[string]string{
		"name": "trim",
	}
}

func (r *FileImageJson) Rules(ctx contractshttp.Context) map[string]string {
	return map[string]string{
		"name":  "required",
		"file":  "file",
		"image": "image",
		"json":  "json",
	}
}

func (r *FileImageJson) Messages(ctx contractshttp.Context) map[string]string {
	return map[string]string{}
}

func (r *FileImageJson) Attributes(ctx contractshttp.Context) map[string]string {
	return map[string]string{}
}

func (r *FileImageJson) PrepareForValidation(ctx contractshttp.Context, data validation.Data) error {
	return nil
}
