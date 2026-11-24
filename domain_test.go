package fiber

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	contractsroute "github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/errors"
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
	route.Post("/", stringHandler("home post"))
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
		{name: "a method the domain doesn't have falls back", method: "POST", host: "admin.example.com", path: "/", expectCode: http.StatusOK, expectBody: "home post"},
		{name: "a method no route has", method: "PUT", host: "admin.example.com", path: "/", expectCode: http.StatusMethodNotAllowed},
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

	route.Domain("Admin.Example.com.").Get("/", stringHandler("admin home")).Name("admin.home")
	route.Get("/", stringHandler("home")).Name("home")

	admin := route.Info("admin.home")
	assert.Equal(t, "admin.example.com", admin.Domain)
	assert.Equal(t, "/", admin.Path)
	assert.Equal(t, "GET|HEAD", admin.Method)

	home := route.Info("home")
	assert.Empty(t, home.Domain)
	assert.Equal(t, "/", home.Path)
	assert.Len(t, route.GetRoutes(), 2)
}

type abortDomainMiddleware struct{}

func (r *abortDomainMiddleware) Signature() string { return "abort_domain" }

func (r *abortDomainMiddleware) Handle(ctx contractshttp.Context) {
	ctx.Request().Abort(http.StatusNonAuthoritativeInfo)
}

func TestDomainRequestInfo(t *testing.T) {
	route := newDomainTestRoute(t)

	infoName := func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, ctx.Request().Info().Name)
	}
	route.Domain("admin.example.com").Middleware(&abortDomainMiddleware{}).
		Get("/", infoName).Name("admin.home").WithoutMiddleware(&abortDomainMiddleware{})
	route.Middleware(&abortDomainMiddleware{}).Get("/", infoName).Name("home")

	// The domain route finds its own Info, so its excluded middleware is skipped.
	code, body := serveHost(t, route, "GET", "admin.example.com", "/")
	assert.Equal(t, http.StatusOK, code)
	assert.Equal(t, "admin.home", body)

	// The route on any host with the same path keeps its middleware.
	code, _ = serveHost(t, route, "GET", "example.com", "/")
	assert.Equal(t, http.StatusNonAuthoritativeInfo, code)
}

func TestDomainInvalid(t *testing.T) {
	route := newDomainTestRoute(t)

	for _, domain := range []string{"", "https://example.com", "example.com:8080", "example.com/admin", "*.example.com"} {
		assert.PanicsWithError(t, errors.RouteInvalidDomain.Args(domain).Error(), func() {
			route.Domain(domain)
		}, domain)
	}
}

func TestDomainParameterRequestInfo(t *testing.T) {
	route := newDomainTestRoute(t)

	route.Domain("{Account}.example.com").Get("/", func(ctx contractshttp.Context) contractshttp.Response {
		info := ctx.Request().Info()
		return ctx.Response().String(http.StatusOK, info.Name+" "+info.Domain+" "+ctx.Request().Route("Account"))
	}).Name("account.home")

	_, body := serveHost(t, route, "GET", "ABC.example.com", "/")
	assert.Equal(t, "account.home {Account}.example.com abc", body)
}

func TestDomainsWithTheSameParameters(t *testing.T) {
	route := newDomainTestRoute(t)

	route.Domain("{account}.example.com").Get("/", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, "account "+ctx.Request().Route("account"))
	}).Name("account.home")
	route.Domain("{tenant}.example.com").Get("/about", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, "tenant "+ctx.Request().Route("tenant")+" "+ctx.Request().Info().Name)
	}).Name("tenant.about")

	_, body := serveHost(t, route, "GET", "acme.example.com", "/")
	assert.Equal(t, "account acme", body)
	_, body = serveHost(t, route, "GET", "acme.example.com", "/about")
	assert.Equal(t, "tenant acme tenant.about", body)
	assert.Equal(t, "{tenant}.example.com", route.Info("tenant.about").Domain)
}

func TestDomainParameterWithThePathParameterName(t *testing.T) {
	route := newDomainTestRoute(t)

	route.Domain("{account}.example.com").Get("/tenants/{account}", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().String(http.StatusOK, ctx.Request().Route("account")+" "+ctx.Request().Input("account")+" "+ctx.Request().All()["account"].(string))
	})

	_, body := serveHost(t, route, "GET", "acme.example.com", "/tenants/path")
	assert.Equal(t, "path path path", body)
}

func TestDomainStatic(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("file"), 0o644))

	counter := &countMiddleware{}
	route := newDomainTestRoute(t, counter)

	route.Domain("admin.example.com").Group(func(router contractsroute.Router) {
		router.Static("/s", dir)
		router.StaticFS("/fs", http.Dir(dir))
		router.StaticFile("/one.txt", filepath.Join(dir, "a.txt"))
	})
	route.StaticFS("/any", http.Dir(dir))
	route.Get("/s/missing.txt", stringHandler("any host"))
	route.Get("/fs/missing.txt", stringHandler("any host"))

	tests := []struct {
		host       string
		path       string
		expectCode int
		expectBody string
	}{
		{host: "admin.example.com", path: "/s/a.txt", expectCode: http.StatusOK, expectBody: "file"},
		{host: "admin.example.com", path: "/fs/a.txt", expectCode: http.StatusOK, expectBody: "file"},
		{host: "admin.example.com", path: "/one.txt", expectCode: http.StatusOK, expectBody: "file"},
		{host: "example.com", path: "/any/a.txt", expectCode: http.StatusOK, expectBody: "file"},
		// A missing file is a 404 of the domain, not a route of any host.
		{host: "admin.example.com", path: "/s/missing.txt", expectCode: http.StatusNotFound},
		{host: "admin.example.com", path: "/fs/missing.txt", expectCode: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.host+test.path, func(t *testing.T) {
			counter.count = 0
			code, body := serveHost(t, route, "GET", test.host, test.path)

			assert.Equal(t, test.expectCode, code)
			if test.expectBody != "" {
				assert.Equal(t, test.expectBody, body)
			}
			assert.Equal(t, 1, counter.count)
		})
	}
}

type authDomainMiddleware struct{}

func (r *authDomainMiddleware) Signature() string { return "auth_domain" }

func (r *authDomainMiddleware) Handle(ctx contractshttp.Context) {
	ctx.Request().Abort(http.StatusUnauthorized)
}

func TestDomainDuplicateRoute(t *testing.T) {
	route := newDomainTestRoute(t)

	admin := route.Domain("admin.example.com").Middleware(&authDomainMiddleware{})
	admin.Get("/", stringHandler("first")).Name("first")
	admin.Get("/other", stringHandler("other"))
	admin.Get("/", stringHandler("second")).Name("second").WithoutMiddleware(&authDomainMiddleware{})

	// Fiber runs the first route, and its Info is kept.
	code, _ := serveHost(t, route, "GET", "admin.example.com", "/")
	assert.Equal(t, http.StatusUnauthorized, code)
	assert.Equal(t, "/", route.Info("first").Path)
	assert.Empty(t, route.Info("second").Path)
}
