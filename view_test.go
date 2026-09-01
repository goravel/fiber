package fiber

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	contractshttp "github.com/goravel/framework/contracts/http"
	foundationjson "github.com/goravel/framework/foundation/json"
	mocksconfig "github.com/goravel/framework/mocks/config"
	mocksview "github.com/goravel/framework/mocks/view"
	"github.com/goravel/framework/session"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestView_Make(t *testing.T) {
	var (
		err        error
		route      *Route
		req        *http.Request
		mockConfig *mocksconfig.Config
		mockView   *mocksview.View
	)

	assert.Nil(t, file.PutContent(path.Resource("views", "empty.tmpl"), `{{ define "empty.tmpl" }}
1
{{ end }}
`))
	assert.Nil(t, file.PutContent(path.Resource("views", "data.tmpl"), `{{ define "data.tmpl" }}
{{ .Name }}
{{ .Age }}
{{ end }}
`))

	defer func() {
		assert.Nil(t, file.Remove(path.Resource()))
	}()

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

		mockView = mocksview.NewView(t)
		ViewFacade = mockView
		mockView.EXPECT().RegisteredViews().Return(nil).Once()
	}
	tests := []struct {
		name        string
		method      string
		url         string
		setup       func(method, url string) error
		expectCode  int
		expectBody  string
		expectPanic bool
	}{
		{
			name:   "data is empty, shared is empty",
			method: "GET",
			url:    "/make/empty",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(nil).Once()

				route.Get("/make/empty", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("empty.tmpl")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\n1\n",
		},
		{
			name:   "data is empty, shared is not empty",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
					"Age":  18,
				}).Once()

				route.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl")
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest\n18\n",
		},
		{
			name:   "data is not empty, shared is not empty",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
				}).Once()

				route.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl", map[string]any{
						"Age": 18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest\n18\n",
		},
		{
			name:   "data is not empty, shared is not empty, and data contains shared key",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
				}).Once()

				route.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl", map[string]any{
						"Name": "test1",
						"Age":  18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest1\n18\n",
		},
		{
			name:   "data is struct, shared is not empty, and data contains shared key",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(map[string]any{
					"Name": "test",
				}).Once()

				route.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().Make("data.tmpl", struct {
						Name string
						Age  int
					}{
						Name: "test1",
						Age:  18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest1\n18\n",
		},
		{
			name:   "data is []string",
			method: "GET",
			url:    "/make/data",
			setup: func(method, url string) error {
				mockView.On("GetShared").Return(nil).Once()

				route.Get("/make/data", func(ctx contractshttp.Context) contractshttp.Response {
					assert.Panics(t, func() {
						ctx.Response().View().Make("data.tmpl", []string{"test"})
					})

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
			expectBody: "",
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

			req.Host = "example.com"
			resp, err := route.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
			mockView.AssertExpectations(t)
		})
	}
}

func TestView_LoadViewsFrom(t *testing.T) {
	// A package/module view directory registered via View.LoadViewsFrom() in a
	// provider's Boot(), i.e. after the route engine is built. The default
	// template set must be compiled lazily (on the first serve/Test call) so
	// that this directory is picked up. See goravel/goravel#989.
	pkgDir, err := os.MkdirTemp("", "goravel-fiber-loadviews-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir))
	}()

	assert.Nil(t, file.PutContent(filepath.Join(pkgDir, "auth.tmpl"), `{{ define "auth.tmpl" }}Hello {{ .name }} from module{{ end }}`))

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
	ConfigFacade = mockConfig

	mockView := mocksview.NewView(t)
	ViewFacade = mockView

	// The route engine is built (init()) before providers boot, and at that
	// point no LoadViewsFrom() call has happened yet. No RegisteredViews()
	// expectation is set before init(): if the default template set were still
	// compiled eagerly inside init(), the mock would be called unexpectedly and
	// the test would fail.
	//
	// Note on fidelity: unlike production (where ViewFacade is nil during init
	// because Boot() runs later), ViewFacade is set before init() here. This is
	// deliberate — it makes the old eager code fail loudly (unexpected
	// RegisteredViews() mock call) instead of silently compiling app views only,
	// which the test would otherwise mistake for success.
	route := &Route{
		config: mockConfig,
		driver: "fiber",
	}
	err = route.init(nil)
	require.Nil(t, err)

	// Provider Boot() registers its module view directory.
	mockView.EXPECT().LoadViewsFrom(pkgDir).Once()
	mockView.LoadViewsFrom(pkgDir)
	mockView.EXPECT().RegisteredViews().Return([]string{pkgDir}).Once()
	mockView.EXPECT().GetShared().Return(map[string]any{"name": "goravel"}).Once()

	route.Get("/auth", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("auth.tmpl")
	})

	req, err := http.NewRequest("GET", "/auth", nil)
	require.Nil(t, err)
	req.Host = "example.com"

	resp, err := route.Test(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Hello goravel from module", string(body))
}

func TestView_LoadViewsFrom_Rebuild(t *testing.T) {
	// After ensureTemplate() has compiled the default views, a re-init() (e.g.
	// Recover/SetGlobalMiddleware triggering an engine rebuild) must not lose
	// them: the lazily-bound holder is reused across rebuilds.
	pkgDir, err := os.MkdirTemp("", "goravel-fiber-loadviews-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir))
	}()

	assert.Nil(t, file.PutContent(filepath.Join(pkgDir, "auth.tmpl"), `{{ define "auth.tmpl" }}Hello {{ .name }} from module{{ end }}`))

	mockConfig := mocksconfig.NewConfig(t)
	mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(nil).Times(4)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Times(2)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Times(2)
	mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Times(2)
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Times(2)
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Times(2)
	mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("X-Forwarded-For").Times(2)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Times(2)
	mockConfig.EXPECT().GetBool("app.debug", false).Return(true).Times(2)
	mockConfig.EXPECT().GetString("app.timezone", "UTC").Return("UTC").Times(2)
	ConfigFacade = mockConfig

	mockView := mocksview.NewView(t)
	ViewFacade = mockView

	route := &Route{
		config: mockConfig,
		driver: "fiber",
	}
	err = route.init(nil)
	require.Nil(t, err)

	// Provider Boot() registers its module view directory.
	mockView.EXPECT().LoadViewsFrom(pkgDir).Once()
	mockView.LoadViewsFrom(pkgDir)
	mockView.EXPECT().RegisteredViews().Return([]string{pkgDir}).Once()
	mockView.EXPECT().GetShared().Return(map[string]any{"name": "goravel"}).Times(2)

	route.Get("/auth", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("auth.tmpl")
	})

	request := func() {
		req, err := http.NewRequest("GET", "/auth", nil)
		require.Nil(t, err)
		req.Host = "example.com"

		resp, err := route.Test(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "Hello goravel from module", string(body))
	}

	// First serve: ensureTemplate() compiles the default template set once.
	request()

	// Engine rebuild: init() runs again, the lazily-bound holder is reused.
	// Routes are registered on the rebuilt engine before serving again.
	//
	// Recover() also reassigns the package-level globalRecoverCallback; restore
	// the previous value so this test leaves no lasting global side effect.
	previousRecoverCallback := globalRecoverCallback
	defer func() {
		globalRecoverCallback = previousRecoverCallback
	}()
	route.Recover(defaultRecoverCallback)
	route.Get("/auth", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("auth.tmpl")
	})
	request()
}

func TestView_First(t *testing.T) {
	var (
		err        error
		route      *Route
		req        *http.Request
		mockConfig *mocksconfig.Config
		mockView   *mocksview.View
	)

	assert.Nil(t, file.PutContent(path.Resource("views", "empty.tmpl"), `{{ define "empty.tmpl" }}
1
{{ end }}
`))
	assert.Nil(t, file.PutContent(path.Resource("views", "data.tmpl"), `{{ define "data.tmpl" }}
{{ .Name }}
{{ .Age }}
{{ end }}
`))

	defer func() {
		assert.Nil(t, file.Remove(path.Resource()))
	}()

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

		mockView = mocksview.NewView(t)
		ViewFacade = mockView
		mockView.EXPECT().RegisteredViews().Return(nil).Once()
	}
	tests := []struct {
		name        string
		method      string
		url         string
		setup       func(method, url string) error
		expectCode  int
		expectBody  string
		expectPanic bool
	}{
		{
			name:   "found the first view",
			method: "GET",
			url:    "/first",
			setup: func(method, url string) error {
				mockView.On("Exists", "empty.tmpl").Return(true).Once()
				mockView.On("GetShared").Return(nil).Once()

				route.Get("/first", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().First([]string{"empty.tmpl", "data.tmpl"})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\n1\n",
		},
		{
			name:   "found the second view",
			method: "GET",
			url:    "/first",
			setup: func(method, url string) error {
				mockView.On("Exists", "empty.tmpl").Return(false).Once()
				mockView.On("Exists", "data.tmpl").Return(true).Once()
				mockView.On("GetShared").Return(nil).Once()

				route.Get("/first", func(ctx contractshttp.Context) contractshttp.Response {
					return ctx.Response().View().First([]string{"empty.tmpl", "data.tmpl"}, map[string]any{
						"Name": "test",
						"Age":  18,
					})
				})

				var err error
				req, err = http.NewRequest(method, url, nil)
				if err != nil {
					return err
				}

				return nil
			},
			expectCode: http.StatusOK,
			expectBody: "\ntest\n18\n",
		},
		{
			name:   "no view found",
			method: "GET",
			url:    "/first",
			setup: func(method, url string) error {
				mockView.On("Exists", "empty.tmpl").Return(false).Once()
				mockView.On("Exists", "data.tmpl").Return(false).Once()

				route.Get("/first", func(ctx contractshttp.Context) contractshttp.Response {
					assert.Panics(t, func() {
						ctx.Response().View().First([]string{"empty.tmpl", "data.tmpl"}, map[string]any{
							"Name": "test",
							"Age":  18,
						})
					})

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
			expectBody: "",
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

			req.Host = "example.com"
			resp, err := route.Test(req)
			assert.NoError(t, err)

			if test.expectBody != "" {
				body, err := io.ReadAll(resp.Body)
				assert.Nil(t, err)
				assert.Equal(t, test.expectBody, string(body))
			}

			assert.Equal(t, test.expectCode, resp.StatusCode)

			mockConfig.AssertExpectations(t)
			mockView.AssertExpectations(t)
		})
	}
}

func TestView_CSRFToken(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "csrf.tmpl"), `{{ define "csrf.tmpl" }}
csrf_token={{ .csrf_token }}
{{ end }}
`))

	defer func() {
		assert.Nil(t, file.Remove(path.Resource()))
	}()

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
	ConfigFacade = mockConfig

	mockView := mocksview.NewView(t)
	ViewFacade = mockView
	mockView.EXPECT().RegisteredViews().Return(nil).Once()
	mockView.EXPECT().GetShared().Return(map[string]any{}).Once()

	t.Run("CSRF token", func(t *testing.T) {
		route := &Route{
			config: mockConfig,
			driver: "fiber",
		}
		err := route.init(nil)
		require.Nil(t, err)

		route.Get("/csrf", func(ctx contractshttp.Context) contractshttp.Response {
			sessionData := session.NewSession("goravel_session", nil, foundationjson.New())
			ctx.Request().SetSession(sessionData)
			err = sessionData.Regenerate()
			assert.Nil(t, err)
			return ctx.Response().View().Make("csrf.tmpl")
		})
		req, err := http.NewRequest("GET", "/csrf", nil)
		assert.Nil(t, err)
		req.Host = "example.com"
		resp, err := route.Test(req)
		body, err := io.ReadAll(resp.Body)
		assert.Nil(t, err)
		assert.Regexp(t, `^\ncsrf_token=([A-Za-z0-9\-_]+)\n$`, string(body))
	})
}

func TestStructToMap(t *testing.T) {
	data := struct {
		Name string
		Age  int
	}{
		Name: "test",
		Age:  18,
	}

	dataMap := structToMap(data)
	assert.Equal(t, "test", dataMap["Name"])
	assert.Equal(t, 18, dataMap["Age"])

	dataMap = structToMap(&data)
	assert.Equal(t, "test", dataMap["Name"])
	assert.Equal(t, 18, dataMap["Age"])
}

func TestFillShared(t *testing.T) {
	shared := map[string]any{
		"Name": "test",
	}
	data := map[string]any{
		"Age": 18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test", data["Name"])
	assert.Equal(t, 18, data["Age"])

	data = map[string]any{
		"Name": "test1",
		"Age":  18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test1", data["Name"])
	assert.Equal(t, 18, data["Age"])

	type Map map[string]any
	data = Map{
		"Age": 18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test", data["Name"])
	assert.Equal(t, 18, data["Age"])

	data = Map{
		"Name": "test1",
		"Age":  18,
	}
	fillShared(data, shared)
	assert.Equal(t, "test1", data["Name"])
	assert.Equal(t, 18, data["Age"])
}
