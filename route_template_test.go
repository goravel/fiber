package fiber

import (
	"errors"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	contractshttp "github.com/goravel/framework/contracts/http"
	mocksconfig "github.com/goravel/framework/mocks/config"
	mocksview "github.com/goravel/framework/mocks/view"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectInit adds the config reads one init() run performs, with debug off so
// no request logger is installed.
func expectInit(mockConfig *mocksconfig.Config, templateConfig any) {
	// init() reads the template key twice: once as an instance, once as a callback.
	templateReads := 2
	if _, ok := templateConfig.(fiber.Views); ok {
		templateReads = 1
	}
	mockConfig.EXPECT().Get("http.drivers.fiber.template").Return(templateConfig).Times(templateReads)
	mockConfig.EXPECT().GetBool("http.drivers.fiber.immutable", true).Return(true).Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.prefork", false).Return(false).Once()
	mockConfig.EXPECT().Get("http.drivers.fiber.trusted_proxies").Return(nil).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.body_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetInt("http.drivers.fiber.header_limit", 4096).Return(4096).Once()
	mockConfig.EXPECT().GetString("http.drivers.fiber.proxy_header", "").Return("").Once()
	mockConfig.EXPECT().GetBool("http.drivers.fiber.enable_trusted_proxy_check", false).Return(false).Once()
	mockConfig.EXPECT().GetBool("app.debug", false).Return(false).Once()
}

// newTemplateTestRoute builds a Route the way NewRoute does, with the given
// http.drivers.fiber.template value, without booting any facade.
func newTemplateTestRoute(t *testing.T, templateConfig any) (*Route, *mocksconfig.Config) {
	t.Helper()

	mockConfig := mocksconfig.NewConfig(t)
	expectInit(mockConfig, templateConfig)
	ConfigFacade = mockConfig

	route := &Route{
		config: mockConfig,
		driver: "fiber",
	}
	route.init(nil)

	return route, mockConfig
}

func serve(t *testing.T, route *Route, url string) (int, string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := route.Test(req)
	require.NoError(t, err)
	defer func() {
		assert.Nil(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(body)
}

// TestRoute_TemplateCallbackDeferred guards the ordering fix for the custom
// template callback: it must not run while the driver is constructed, because
// that can happen before any service provider has booted. It runs once, from
// CompileViews() or the first serve call, and therefore sees views and functions
// registered during Boot.
func TestRoute_TemplateCallbackDeferred(t *testing.T) {
	pkgDir := path.Resource("pkg_deferred")
	require.NoError(t, os.MkdirAll(pkgDir, os.ModePerm))
	require.NoError(t, file.PutContent(path.Resource("pkg_deferred", "shout.tmpl"), `{{ define "shout.tmpl" }}{{ upper .word }}{{ end }}`))
	defer func() {
		ConfigFacade = nil
		ViewFacade = nil
		assert.Nil(t, file.Remove(path.Resource()))
	}()

	calls := 0
	callback := func() (fiber.Views, error) {
		calls++

		// NewTemplate reads the registry itself, so the callback only works if it
		// runs after the package registered its views.
		return NewTemplate(RenderOptions{
			FuncMap: template.FuncMap{"upper": strings.ToUpper},
		})
	}

	route, mockConfig := newTemplateTestRoute(t, callback)
	assert.Equal(t, 0, calls, "init() must not invoke the template callback")
	require.NotNil(t, route.templateHolder)
	assert.Nil(t, route.templateHolder.views)

	// A provider's Boot() registers the package views after the driver exists.
	mockView := mocksview.NewView(t)
	mockView.EXPECT().RegisteredViews().Return([]string{pkgDir}).Once()
	mockView.EXPECT().RegisteredViewFS().Return(nil).Once()
	mockView.EXPECT().Funcs().Return(nil).Once()
	mockView.EXPECT().GetShared().Return(nil).Times(3)
	ViewFacade = mockView

	route.Get("/shout", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("shout.tmpl", map[string]any{"word": "hi"})
	})

	// The framework compiles after Boot through the view.Compiler bridge.
	require.NoError(t, route.CompileViews())
	assert.Equal(t, 1, calls)

	status, body := serve(t, route, "/shout")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "HI", body)
	assert.Equal(t, 1, calls, "serving must not compile again")

	// Rebuilding the engine after the first serve keeps the compiled views and
	// does not invoke the callback a second time.
	expectInit(mockConfig, callback)
	route.GlobalMiddleware()
	route.Get("/shout", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("shout.tmpl", map[string]any{"word": "again"})
	})
	status, body = serve(t, route, "/shout")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "AGAIN", body)
	assert.Equal(t, 1, calls)

	// Without the bridge the first serve call compiles; the result is identical.
	route2, _ := newTemplateTestRoute(t, callback)
	route2.Get("/shout", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("shout.tmpl", map[string]any{"word": "late"})
	})
	mockView.EXPECT().RegisteredViews().Return([]string{pkgDir}).Once()
	mockView.EXPECT().RegisteredViewFS().Return(nil).Once()
	mockView.EXPECT().Funcs().Return(nil).Once()
	status, body = serve(t, route2, "/shout")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "LATE", body)
	assert.Equal(t, 2, calls)
}

// TestRoute_TemplateCallbackError guards that a failing callback no longer aborts
// driver construction, and that the error is sticky across calls instead of
// being reported once and then forgotten.
func TestRoute_TemplateCallbackError(t *testing.T) {
	defer func() {
		ConfigFacade = nil
	}()

	calls := 0
	route, _ := newTemplateTestRoute(t, func() (fiber.Views, error) {
		calls++

		return nil, errors.New("callback failed")
	})

	err := route.CompileViews()
	require.EqualError(t, err, "callback failed")
	assert.Equal(t, 1, calls)

	// Sticky: later calls report the same error without invoking the callback again.
	assert.Equal(t, err, route.ensureTemplate())
	assert.Equal(t, 1, calls)

	req, reqErr := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, reqErr)
	resp, testErr := route.Test(req)
	assert.Nil(t, resp)
	assert.Equal(t, err, testErr)
}

// TestRoute_TemplateCallbackNilViews guards that a callback returning no views
// falls back to the default set instead of leaving the holder empty, which would
// fail every render after a successful CompileViews().
func TestRoute_TemplateCallbackNilViews(t *testing.T) {
	require.NoError(t, os.MkdirAll(path.Resource("views"), os.ModePerm))
	require.NoError(t, file.PutContent(path.Resource("views", "home.tmpl"), `{{ define "home.tmpl" }}Home{{ end }}`))
	defer func() {
		ConfigFacade = nil
		ViewFacade = nil
		assert.Nil(t, file.Remove(path.Resource()))
	}()

	calls := 0
	route, _ := newTemplateTestRoute(t, func() (fiber.Views, error) {
		calls++

		return nil, nil
	})

	mockView := mocksview.NewView(t)
	mockView.EXPECT().RegisteredViews().Return(nil).Once()
	mockView.EXPECT().RegisteredViewFS().Return(nil).Once()
	mockView.EXPECT().Funcs().Return(nil).Once()
	mockView.EXPECT().GetShared().Return(nil).Once()
	ViewFacade = mockView

	require.NoError(t, route.CompileViews())
	assert.Equal(t, 1, calls)
	require.NotNil(t, route.templateHolder.views)

	route.Get("/home", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("home.tmpl")
	})
	status, body := serve(t, route, "/home")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Home", body)
	assert.Equal(t, 1, calls)
}

// TestRoute_DefaultTemplateStickyError guards that a DefaultTemplate() failure is
// returned on every serve call, not only the first.
func TestRoute_DefaultTemplateStickyError(t *testing.T) {
	require.NoError(t, os.MkdirAll(path.Resource("views"), os.ModePerm))
	require.NoError(t, file.PutContent(path.Resource("views", "bad.tmpl"), `{{ define "bad.tmpl" }}{{`))
	defer func() {
		ConfigFacade = nil
		ViewFacade = nil
		assert.Nil(t, file.Remove(path.Resource()))
	}()

	route, _ := newTemplateTestRoute(t, nil)

	mockView := mocksview.NewView(t)
	mockView.EXPECT().RegisteredViews().Return(nil).Once()
	mockView.EXPECT().RegisteredViewFS().Return(nil).Once()
	mockView.EXPECT().Funcs().Return(nil).Once()
	ViewFacade = mockView

	first := route.CompileViews()
	require.Error(t, first)
	assert.Contains(t, first.Error(), "failed to parse view")
	assert.Equal(t, first, route.ensureTemplate())

	req, err := http.NewRequest(http.MethodGet, "/", nil)
	require.NoError(t, err)
	_, err = route.Test(req)
	assert.Equal(t, first, err)
}

// TestRoute_DefaultTemplateUsesViewFuncs guards that functions registered on the
// view facade (View.AddFuncs, which ApplicationBuilder.WithViewFuncs feeds) are
// installed on the default template set.
func TestRoute_DefaultTemplateUsesViewFuncs(t *testing.T) {
	require.NoError(t, os.MkdirAll(path.Resource("views"), os.ModePerm))
	require.NoError(t, file.PutContent(path.Resource("views", "greet.tmpl"), `{{ define "greet.tmpl" }}{{ greet "Ada" }}{{ end }}`))
	defer func() {
		ConfigFacade = nil
		ViewFacade = nil
		assert.Nil(t, file.Remove(path.Resource()))
	}()

	route, _ := newTemplateTestRoute(t, nil)

	mockView := mocksview.NewView(t)
	mockView.EXPECT().RegisteredViews().Return(nil).Once()
	mockView.EXPECT().RegisteredViewFS().Return(nil).Once()
	mockView.EXPECT().Funcs().Return(template.FuncMap{
		"greet": func(name string) string { return "Hello, " + name + "!" },
	}).Once()
	mockView.EXPECT().GetShared().Return(nil).Once()
	ViewFacade = mockView

	route.Get("/greet", func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("greet.tmpl")
	})

	status, body := serve(t, route, "/greet")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Hello, Ada!", body)
}

// TestRoute_CompileViews guards the view.Compiler contract: CompileViews compiles
// the default set once, later serve calls reuse it, and an engine rebuild after
// the compile keeps the views.
func TestRoute_CompileViews(t *testing.T) {
	require.NoError(t, os.MkdirAll(path.Resource("views"), os.ModePerm))
	require.NoError(t, file.PutContent(path.Resource("views", "home.tmpl"), `{{ define "home.tmpl" }}Home{{ end }}`))
	defer func() {
		ConfigFacade = nil
		ViewFacade = nil
		assert.Nil(t, file.Remove(path.Resource()))
	}()

	route, mockConfig := newTemplateTestRoute(t, nil)

	mockView := mocksview.NewView(t)
	mockView.EXPECT().RegisteredViews().Return(nil).Once()
	mockView.EXPECT().RegisteredViewFS().Return(nil).Once()
	mockView.EXPECT().Funcs().Return(nil).Once()
	mockView.EXPECT().GetShared().Return(nil).Twice()
	ViewFacade = mockView

	require.NoError(t, route.CompileViews())
	require.NotNil(t, route.templateHolder.views)
	require.NoError(t, route.CompileViews(), "a second call is a no-op")

	handler := func(ctx contractshttp.Context) contractshttp.Response {
		return ctx.Response().View().Make("home.tmpl")
	}
	route.Get("/home", handler)
	status, body := serve(t, route, "/home")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Home", body)

	// SetGlobalMiddleware rebuilds the engine (and drops the routes, by design);
	// the compiled views must survive the rebuild.
	expectInit(mockConfig, nil)
	route.SetGlobalMiddleware(nil)
	require.NotNil(t, route.templateHolder.views)
	route.Get("/home", handler)
	status, body = serve(t, route, "/home")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Home", body)
}

// TestRoute_TemplateInstance guards that a ready fiber.Views configured as an
// instance is bound directly, without a holder, and CompileViews has nothing to do.
func TestRoute_TemplateInstance(t *testing.T) {
	defer func() {
		ConfigFacade = nil
	}()

	instance, err := NewTemplate(RenderOptions{})
	require.NoError(t, err)

	route, _ := newTemplateTestRoute(t, instance)
	assert.Nil(t, route.templateHolder)
	require.NoError(t, route.CompileViews())
	assert.Nil(t, route.templateHolder)
}
