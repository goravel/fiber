package fiber

import (
	"html/template"
	"io"
	"io/fs"
	"maps"
	"sync"

	"github.com/gofiber/fiber/v3"
	contractsview "github.com/goravel/framework/contracts/view"
	frameworkview "github.com/goravel/framework/view"
)

// Delims represents custom delimiters for template actions.
type Delims struct {
	Left  string
	Right string
}

// RenderOptions configures template parsing and rendering.
type RenderOptions struct {
	Delims  *Delims
	FuncMap template.FuncMap
	// ExtraPaths are package view directories in precedence order. nil means the
	// view facade's RegisteredViews(); pass an empty slice to compile none.
	ExtraPaths []string
	// ExtraFS are package view filesystems in precedence order. nil means the
	// view facade's RegisteredViewFS(); pass an empty slice to compile none.
	ExtraFS []fs.FS
}

// Template implements fiber.Views for multi-directory template loading.
// Templates are loaded from the app's resources/views first, then from
// registered package directories. App-defined templates take priority;
// collisions between packages log a warning.
type Template struct {
	mu     sync.RWMutex
	engine *template.Template
}

// NewTemplate creates a Template by parsing .tmpl files from the app views
// directory, the package directories and the package filesystems. The sources
// default to the ones registered on the view facade, and the registered
// functions are installed first with options.FuncMap on top, so a
// http.drivers.<driver>.template callback keeps package views and
// WithViewFuncs without passing them. Precedence, naming and collision handling
// are those of view.Compile. If no files are found, the Template is still valid
// but Render will return an error for any template name.
func NewTemplate(options RenderOptions) (*Template, error) {
	compileOptions := frameworkview.CompileOptions{
		Paths:      options.ExtraPaths,
		FS:         options.ExtraFS,
		Extensions: []string{".tmpl"},
		Log:        LogFacade,
	}
	if options.Delims != nil {
		compileOptions.Delims = &frameworkview.Delims{Left: options.Delims.Left, Right: options.Delims.Right}
	}
	if viewFacade := resolveViewFacade(); viewFacade != nil {
		if compileOptions.Paths == nil {
			compileOptions.Paths = viewFacade.RegisteredViews()
		}
		if compileOptions.FS == nil {
			compileOptions.FS = viewFacade.RegisteredViewFS()
		}
		compileOptions.FuncMap = viewFacade.Funcs()
	}
	if len(options.FuncMap) > 0 {
		if compileOptions.FuncMap == nil {
			compileOptions.FuncMap = make(template.FuncMap, len(options.FuncMap))
		}
		maps.Copy(compileOptions.FuncMap, options.FuncMap)
	}

	instance, err := frameworkview.Compile(compileOptions)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return &Template{}, nil
	}

	return &Template{engine: instance}, nil
}

// DefaultTemplate creates a Template from every source and function registered
// on the view facade. Returns a valid fiber.Views even when there are no
// template files.
func DefaultTemplate() (fiber.Views, error) {
	return NewTemplate(RenderOptions{})
}

// resolveViewFacade returns the view facade ServiceProvider.Boot assigned, or
// resolves it from the container when the driver is used before Boot. nil when
// neither is available, in which case only the application's views compile.
func resolveViewFacade() contractsview.View {
	if ViewFacade != nil {
		return ViewFacade
	}
	if App != nil {
		return App.MakeView()
	}

	return nil
}

// Compile-time assertion that viewsHolder satisfies fiber.Views.
var _ fiber.Views = (*viewsHolder)(nil)

// viewsHolder is a lazily-bound fiber.Views. fiber v3 reads Config.Views from
// its internal config copy at render time and app.Config() returns a copy, so
// the value bound at fiber.New() time cannot be swapped afterwards. This
// indirection lets the template set be compiled later, after all providers
// have booted, and handed to the already-created fiber app (see
// goravel/goravel#989).
type viewsHolder struct {
	mu    sync.RWMutex
	views fiber.Views
}

// set binds the compiled views. Called by ensureTemplate() before the first
// request is served.
func (h *viewsHolder) set(views fiber.Views) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.views = views
}

// Load is a no-op until the views have been compiled.
func (h *viewsHolder) Load() error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.views == nil {
		return nil
	}
	return h.views.Load()
}

// Render delegates to the compiled views. Returns fiber.ErrInternalServerError
// if ensureTemplate() has not run yet.
func (h *viewsHolder) Render(w io.Writer, name string, binding any, layouts ...string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.views == nil {
		return fiber.ErrInternalServerError
	}
	return h.views.Render(w, name, binding, layouts...)
}

// Load is a no-op because template parsing happens eagerly in NewTemplate.
// This method exists solely to satisfy the fiber.Views interface.
func (m *Template) Load() error {
	return nil
}

// Render executes the named template, writing the result to w. If layouts
// are provided, the first layout is rendered instead, with the named
// template available via {{ template }}. Returns fiber.ErrInternalServerError
// if no templates were loaded.
func (m *Template) Render(w io.Writer, name string, data any, layouts ...string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.engine == nil {
		return fiber.ErrInternalServerError
	}

	if len(layouts) > 0 {
		return m.engine.ExecuteTemplate(w, layouts[0], data)
	}

	return m.engine.ExecuteTemplate(w, name, data)
}
