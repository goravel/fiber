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
	"github.com/stretchr/testify/suite"
)

type RouteTestSuite struct {
	suite.Suite
	mockConfig *mocksconfig.Config
	route      *Route
}

func TestRouteTestSuite(t *testing.T) {
	suite.Run(t, new(RouteTestSuite))
}

func (s *RouteTestSuite) SetupTest() {
	routes = make(map[string]map[string]contractshttp.Info)

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

func (s *RouteTestSuite) TestRecoverWithCustomCallback() {
	globalRecoverCallback := func(ctx contractshttp.Context, err any) {
		ctx.Request().Abort(http.StatusInternalServerError)
	}

	s.route.Recover(globalRecoverCallback)

	s.route.Get("/recover", func(ctx contractshttp.Context) contractshttp.Response {
		panic(1)
	})

	req := httptest.NewRequest("GET", "/recover", nil)
	resp, err := s.route.Test(req)
	s.NoError(err)

	body, err := io.ReadAll(resp.Body)
	s.NoError(err)
	s.Equal("Internal Server Error", string(body))
	s.Equal(http.StatusInternalServerError, resp.StatusCode)
}

func (s *RouteTestSuite) TestFallback() {
	s.route.Fallback(func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusNotFound, "not found")
	})
	s.route.Get("/test", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, "test")
	})

	req, err := http.NewRequest("GET", "/test", nil)
	s.NoError(err)
	resp, err := s.route.Test(req)
	s.NoError(err)

	s.Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	s.NoError(err)
	s.Equal("test", string(body))

	req, err = http.NewRequest("GET", "/not-found", nil)
	s.NoError(err)

	resp, err = s.route.Test(req)
	s.NoError(err)

	s.Equal(http.StatusNotFound, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	s.NoError(err)
	s.Equal("not found", string(body))
}

func (s *RouteTestSuite) TestGetRoutes() {
	s.route.Get("/b/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(200, "ok")
	})
	s.route.Post("/b/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(200, "ok")
	})
	s.route.Get("/a/{id}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(200, "ok")
	})

	routes := s.route.GetRoutes()
	s.Len(routes, 3)
	s.Equal("GET|HEAD", routes[0].Method)
	s.Equal("/a/{id}", routes[0].Path)
	s.Equal("GET|HEAD", routes[1].Method)
	s.Equal("/b/{id}", routes[1].Path)
	s.Equal("POST", routes[2].Method)
	s.Equal("/b/{id}", routes[2].Path)
}

func (s *RouteTestSuite) TestGlobalMiddleware() {
	// has timeout middleware
	s.mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
	s.mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()
	s.mockConfig.EXPECT().GetInt("http.request_timeout", 3).Return(1).Once()
	s.route.GlobalMiddleware()
	s.Equal(s.route.instance.HandlersCount(), uint32(4))

	// no timeout middleware
	s.SetupTest()
	s.mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Once()
	s.mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Once()
	s.mockConfig.EXPECT().GetInt("http.request_timeout", 3).Return(0).Once()
	s.route.GlobalMiddleware()
	s.Equal(s.route.instance.HandlersCount(), uint32(3))
}

func (s *RouteTestSuite) TestListen() {
	host := "127.0.0.1:3100"

	s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(200, contractshttp.Json{
			"Hello": "Goravel",
		})
	})

	s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

	go func() {
		l, err := net.Listen("tcp", host)
		s.NoError(err)
		s.NoError(s.route.Listen(l))
	}()

	time.Sleep(1 * time.Second)

	resp, err := http.Get("http://" + host)
	s.NoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	s.NoError(err)
	s.Equal("{\"Hello\":\"Goravel\"}", string(body))
}

func (s *RouteTestSuite) TestListenTLS() {
	host := "127.0.0.1:3101"

	s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(200, contractshttp.Json{
			"Hello": "Goravel",
		})
	})

	s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
	s.mockConfig.EXPECT().GetString("http.tls.ssl.cert").Return("test_ca.crt").Once()
	s.mockConfig.EXPECT().GetString("http.tls.ssl.key").Return("test_ca.key").Once()

	go func() {
		l, err := net.Listen("tcp", host)
		s.NoError(err)
		s.NoError(s.route.ListenTLS(l))
	}()

	time.Sleep(1 * time.Second)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get("https://" + host)
	s.NoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	s.NoError(err)
	s.Equal("{\"Hello\":\"Goravel\"}", string(body))
}

func (s *RouteTestSuite) TestListenTLSWithCert() {
	host := "127.0.0.1:3102"

	s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(200, contractshttp.Json{
			"Hello": "Goravel",
		})
	})

	s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

	go func() {
		l, err := net.Listen("tcp", host)
		s.NoError(err)
		s.NoError(s.route.ListenTLSWithCert(l, "test_ca.crt", "test_ca.key"))
	}()

	time.Sleep(1 * time.Second)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get("https://" + host)
	s.NoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	s.NoError(err)
	s.Equal("{\"Hello\":\"Goravel\"}", string(body))
}

func (s *RouteTestSuite) TestInfo() {
	action := s.route.Get("/test", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().Json(200, contractshttp.Json{
			"Hello": "Goravel",
		})
	}).Name("test")

	s.Equal(&Action{
		method: "GET|HEAD",
		path:   "/test",
	}, action)

	info := s.route.Info("test")
	s.Equal("GET|HEAD", info.Method)
	s.Equal("test", info.Name)
	s.Equal("/test", info.Path)
}

func (s *RouteTestSuite) TestRun() {
	s.Run("error when default port is empty", func() {
		s.mockConfig.EXPECT().GetString("http.host").Return("127.0.0.1").Once()
		s.mockConfig.EXPECT().GetString("http.port").Return("").Once()

		s.Equal(errors.New("port can't be empty"), s.route.Run())
	})

	s.Run("use default host", func() {
		host := "127.0.0.1"
		port := "3031"
		addr := host + ":" + port

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Json(200, contractshttp.Json{
				"Hello": "Goravel",
			})
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
		s.mockConfig.EXPECT().GetString("http.host").Return(host).Once()
		s.mockConfig.EXPECT().GetString("http.port").Return(port).Once()

		go func() {
			s.NoError(s.route.Run())
		}()

		time.Sleep(1 * time.Second)

		hostUrl := "http://" + addr

		resp, err := http.Get(hostUrl)
		s.NoError(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		s.Equal("{\"Hello\":\"Goravel\"}", string(body))
	})

	s.Run("use custom host", func() {
		host := "127.0.0.1"
		port := "3032"
		addr := host + ":" + port

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Json(200, contractshttp.Json{
				"Hello": "Goravel",
			})
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

		go func() {
			s.NoError(s.route.Run(addr))
		}()

		time.Sleep(1 * time.Second)

		hostUrl := "http://" + addr

		resp, err := http.Get(hostUrl)
		s.NoError(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		s.Equal("{\"Hello\":\"Goravel\"}", string(body))
	})
}

func (s *RouteTestSuite) TestRunTLS() {
	s.Run("error when default port is empty", func() {
		s.mockConfig.EXPECT().GetString("http.tls.host").Return("127.0.0.1").Once()
		s.mockConfig.EXPECT().GetString("http.tls.port").Return("").Once()

		s.Equal(errors.New("port can't be empty"), s.route.RunTLS())
	})

	s.Run("use default host", func() {
		host := "127.0.0.1"
		port := "3033"
		addr := host + ":" + port

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Json(200, contractshttp.Json{
				"Hello": "Goravel",
			})
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
		s.mockConfig.EXPECT().GetString("http.tls.host").Return(host).Once()
		s.mockConfig.EXPECT().GetString("http.tls.port").Return(port).Once()
		s.mockConfig.EXPECT().GetString("http.tls.ssl.cert").Return("test_ca.crt").Once()
		s.mockConfig.EXPECT().GetString("http.tls.ssl.key").Return("test_ca.key").Once()

		go func() {
			s.NoError(s.route.RunTLS())
		}()

		time.Sleep(1 * time.Second)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		resp, err := client.Get("https://" + addr)
		s.NoError(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		s.Equal("{\"Hello\":\"Goravel\"}", string(body))
	})

	s.Run("use custom host", func() {
		host := "127.0.0.1"
		port := "3034"
		addr := host + ":" + port

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Json(200, contractshttp.Json{
				"Hello": "Goravel",
			})
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
		s.mockConfig.EXPECT().GetString("http.tls.ssl.cert").Return("test_ca.crt").Once()
		s.mockConfig.EXPECT().GetString("http.tls.ssl.key").Return("test_ca.key").Once()

		go func() {
			s.NoError(s.route.RunTLS(addr))
		}()

		time.Sleep(1 * time.Second)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		resp, err := client.Get("https://" + addr)
		s.NoError(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		s.Equal("{\"Hello\":\"Goravel\"}", string(body))
	})
}

func (s *RouteTestSuite) TestRunTLSWithCert() {
	s.Run("error when default host is empty", func() {
		s.Equal(errors.New("host can't be empty"), s.route.RunTLSWithCert("", "test_ca.crt", "test_ca.key"))
	})

	s.Run("use default host", func() {
		host := "127.0.0.1"
		port := "3035"
		addr := host + ":" + port

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Json(200, contractshttp.Json{
				"Hello": "Goravel",
			})
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

		go func() {
			s.NoError(s.route.RunTLSWithCert(addr, "test_ca.crt", "test_ca.key"))
		}()

		time.Sleep(1 * time.Second)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get("https://" + addr)
		s.NoError(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		s.Equal("{\"Hello\":\"Goravel\"}", string(body))
	})

	s.Run("use custom host", func() {
		host := "127.0.0.1"
		port := "3036"
		addr := host + ":" + port

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Json(200, contractshttp.Json{
				"Hello": "Goravel",
			})
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()

		go func() {
			s.NoError(s.route.RunTLSWithCert(addr, "test_ca.crt", "test_ca.key"))
		}()

		time.Sleep(1 * time.Second)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get("https://" + addr)
		s.NoError(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		s.NoError(err)
		s.Equal("{\"Hello\":\"Goravel\"}", string(body))
	})
}

func (s *RouteTestSuite) TestNewRoute() {
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
				mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
				mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
				mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
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
				mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
				mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
				mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
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
				mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
				mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
				mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
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
		s.Run(test.name, func() {
			mockConfig = mocksconfig.NewConfig(s.T())
			test.setup()
			route, err := NewRoute(mockConfig, test.parameters)
			s.Equal(test.expectError, err)
			if route != nil {
				s.IsType(test.expectTemplate, route.instance.Config().Views)
			}
		})
	}
}

func (s *RouteTestSuite) TestShutdown() {
	s.Run("no new requests will be accepted after shutdown", func() {
		host := "127.0.0.1"
		port := "6789"
		addr := fmt.Sprintf("http://%s:%s", host, port)

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().Success().String("Goravel")
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
		s.mockConfig.EXPECT().GetString("http.host").Return(host).Once()
		s.mockConfig.EXPECT().GetString("http.port").Return(port).Once()

		go func() {
			s.NoError(s.route.Run())
		}()

		time.Sleep(1 * time.Second)

		assertHttpNormal(s.T(), addr, true)

		s.NoError(s.route.Shutdown())
		assertHttpNormal(s.T(), addr, false)
	})

	s.Run("ensure that received requests are processed", func() {
		s.SetupTest()

		var (
			host = "127.0.0.1"
			port = "6789"
			addr = fmt.Sprintf("http://%s:%s", host, port)

			count atomic.Int64
		)

		s.route.Get("/", func(ctx contractshttp.Context) contractshttp.Response {
			defer count.Add(1)
			return ctx.Response().Success().String("Goravel")
		})

		s.mockConfig.EXPECT().GetBool("app.debug").Return(true).Once()
		s.mockConfig.EXPECT().GetString("http.host").Return(host).Once()
		s.mockConfig.EXPECT().GetString("http.port").Return(port).Once()

		go func() {
			s.NoError(s.route.Run())
		}()

		time.Sleep(1 * time.Second)

		wg := sync.WaitGroup{}
		count.Store(0)
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				assertHttpNormal(s.T(), addr, true)
			}()
		}

		time.Sleep(100 * time.Millisecond)

		s.NoError(s.route.Shutdown())
		assertHttpNormal(s.T(), addr, false)
		wg.Wait()
		s.Equal(count.Load(), int64(3))
	})
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
