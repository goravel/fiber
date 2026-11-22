package fiber

import (
	"io"
	"net/http"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
	mocksconfig "github.com/goravel/framework/mocks/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countMiddleware struct {
	count int
}

func (r *countMiddleware) Signature() string { return "count" }

func (r *countMiddleware) Handle(ctx contractshttp.Context) {
	r.count++
	ctx.Request().Next()
}

func newDomainTestRoute(t *testing.T, globalMiddleware ...contractshttp.Middleware) *Route {
	routes = make(map[string]map[string]contractshttp.Info)

	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Twice()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("app.debug", false).Return(false).Once()

	route := &Route{
		config: mockConfig,
		driver: "fiber",
	}
	route.init(globalMiddleware)

	return route
}

func stringHandler(body string) contractshttp.HandlerFunc {
	return func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, body)
	}
}

func serveHost(t *testing.T, route *Route, method, host, path string) (int, string) {
	t.Helper()

	request, err := http.NewRequest(method, "http://"+host+path, nil)
	require.NoError(t, err)

	response, err := route.Test(request)
	require.NoError(t, err)
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return response.StatusCode, string(body)
}

func TestDomain(t *testing.T) {
	route := newDomainTestRoute(t)

	route.Domain("admin.example.com").Group(func(router contractsroute.Router) {
		router.Get("/", stringHandler("admin home"))
		router.Prefix("users").Get("/{id}", func(ctx contractshttp.Context) contractshttp.Response {
			return ctx.Response().String(http.StatusOK, "admin user "+ctx.Request().Route("id"))
		})
	})
	route.Domain("{account}.example.com").Get("/", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, "account "+ctx.Request().Route("account")+" "+ctx.Request().Input("account"))
	})
	route.Get("/", stringHandler("home"))
	route.Get("/about", stringHandler("about"))

	tests := []struct {
		name       string
		method     string
		host       string
		path       string
		expectCode int
		expectBody string
	}{
		{name: "the domain's route", method: "GET", host: "admin.example.com", path: "/", expectCode: http.StatusOK, expectBody: "admin home"},
		{name: "with a port and in upper case", method: "GET", host: "Admin.Example.com:3000", path: "/", expectCode: http.StatusOK, expectBody: "admin home"},
		{name: "with a path parameter", method: "GET", host: "admin.example.com", path: "/users/1", expectCode: http.StatusOK, expectBody: "admin user 1"},
		{name: "HEAD on the domain", method: "HEAD", host: "admin.example.com", path: "/", expectCode: http.StatusOK},
		{name: "with a domain parameter", method: "GET", host: "goravel.example.com", path: "/", expectCode: http.StatusOK, expectBody: "account goravel goravel"},
		{name: "a parameter matches one label", method: "GET", host: "a.b.example.com", path: "/", expectCode: http.StatusOK, expectBody: "home"},
		{name: "another host uses the routes without a domain", method: "GET", host: "example.com", path: "/", expectCode: http.StatusOK, expectBody: "home"},
		{name: "a path the domain doesn't have falls back to any host", method: "GET", host: "admin.example.com", path: "/about", expectCode: http.StatusOK, expectBody: "about"},
		{name: "a domain route is not on other hosts", method: "GET", host: "example.com", path: "/users/1", expectCode: http.StatusNotFound},
		{name: "not found on the domain", method: "GET", host: "admin.example.com", path: "/missing", expectCode: http.StatusNotFound},
		{name: "a method the domain doesn't have falls back", method: "POST", host: "admin.example.com", path: "/", expectCode: http.StatusMethodNotAllowed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			code, body := serveHost(t, route, test.method, test.host, test.path)

			assert.Equal(t, test.expectCode, code)
			if test.expectBody != "" {
				assert.Equal(t, test.expectBody, body)
			}
		})
	}
}

func TestDomainRunsTheGlobalMiddlewareOnce(t *testing.T) {
	counter := &countMiddleware{}
	route := newDomainTestRoute(t, counter)

	route.Domain("admin.example.com").Get("/", stringHandler("admin home"))
	route.Get("/about", stringHandler("about"))

	_, body := serveHost(t, route, "GET", "admin.example.com", "/")
	assert.Equal(t, "admin home", body)
	assert.Equal(t, 1, counter.count)

	// Passed to the routes of any host.
	_, body = serveHost(t, route, "GET", "admin.example.com", "/about")
	assert.Equal(t, "about", body)
	assert.Equal(t, 2, counter.count)
}

func TestDomainWithFallback(t *testing.T) {
	route := newDomainTestRoute(t)

	route.Domain("admin.example.com").Get("/", stringHandler("admin home"))
	route.Fallback(stringHandler("fallback"))

	_, body := serveHost(t, route, "GET", "admin.example.com", "/missing")
	assert.Equal(t, "fallback", body)
}

func TestDomainRoutesInfo(t *testing.T) {
	route := newDomainTestRoute(t)

	route.Domain("admin.example.com").Get("/", stringHandler("admin home")).Name("admin.home")
	route.Get("/", stringHandler("home")).Name("home")

	assert.Equal(t, contractshttp.Info{
		Domain:  "admin.example.com",
		Handler: "github.com/goravel/fiber.stringHandler.func1",
		Method:  "GET|HEAD",
		Name:    "admin.home",
		Path:    "/",
	}, route.Info("admin.home"))
	assert.Equal(t, contractshttp.Info{
		Handler: "github.com/goravel/fiber.stringHandler.func1",
		Method:  "GET|HEAD",
		Name:    "home",
		Path:    "/",
	}, route.Info("home"))
	assert.Len(t, route.GetRoutes(), 2)
}

func TestDomainRouterMatch(t *testing.T) {
	router := newDomainRouter("{account}.{region}.Example.com", nil)

	params, ok := router.match("goravel.eu.example.com")
	assert.True(t, ok)
	assert.Equal(t, map[string]string{"account": "goravel", "region": "eu"}, params)

	_, ok = router.match("eu.example.com")
	assert.False(t, ok)
	_, ok = router.match("goravel.eu.example.org")
	assert.False(t, ok)

	static := newDomainRouter("Admin.Example.com", nil)
	_, ok = static.match("admin.example.com")
	assert.True(t, ok)
	_, ok = static.match("adminXexample.com")
	assert.False(t, ok)
}
