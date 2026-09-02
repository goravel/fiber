package fiber

import (
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
)

var defineRe = regexp.MustCompile(`\{\{\s*define\s+"([^"]+)"`)
var customDefineRes = &sync.Map{}

// Delims represents custom delimiters for template actions.
type Delims struct {
	Left  string
	Right string
}

// RenderOptions configures template parsing and rendering.
type RenderOptions struct {
	Delims     *Delims
	FuncMap    template.FuncMap
	ExtraPaths []string
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
// directory and any extra paths. Templates without a {{ define }} block are
// skipped. If no files are found, the Template is still valid but Render
// will return an error for any template name.
func NewTemplate(options RenderOptions) (*Template, error) {
	instance := template.New("")
	if options.Delims != nil {
		instance.Delims(options.Delims.Left, options.Delims.Right)
	}
	if options.FuncMap != nil {
		instance.Funcs(options.FuncMap)
	}

	leftDelim := "{{"
	if options.Delims != nil {
		leftDelim = options.Delims.Left
	}

	appDefines := make(map[string]string)
	pkgDefines := make(map[string]string)
	var files []string

	appDir := path.Resource("views")
	if file.Exists(appDir) {
		if err := walkTmplFiles(appDir, leftDelim, func(filePath string, name string) {
			appDefines[name] = filePath
			files = append(files, filePath)
		}); err != nil {
			return nil, err
		}
	}

	for _, dir := range options.ExtraPaths {
		if !file.Exists(dir) {
			continue
		}
		if err := walkTmplFiles(dir, leftDelim, func(filePath string, name string) {
			if _, ok := appDefines[name]; ok {
				return
			}
			if existing, ok := pkgDefines[name]; ok {
				if LogFacade != nil {
					LogFacade.Warningf("view collision: %q defined in %q and %q, using first", name, existing, filePath)
				}
				return
			}
			pkgDefines[name] = filePath
			files = append(files, filePath)
		}); err != nil {
			return nil, err
		}
	}

	if len(files) == 0 {
		return &Template{}, nil
	}

	tmpl, err := instance.ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return &Template{engine: tmpl}, nil
}

// DefaultTemplate creates a Template with package view directories from
// ViewFacade.RegisteredViews(). Returns a valid fiber.Views even when there
// are no template files.
func DefaultTemplate() (fiber.Views, error) {
	options := RenderOptions{}
	viewFacade := ViewFacade
	// Defensive net only: in normal operation ViewFacade is set by the driver's
	// ServiceProvider.Boot() before the first Run/Listen/Test, so this fallback
	// is not exercised. It exists purely so a mis-wired app (view provider never
	// booted) degrades to "compile app views only" instead of panicking.
	if viewFacade == nil && App != nil {
		viewFacade = App.MakeView()
	}
	if viewFacade != nil {
		options.ExtraPaths = viewFacade.RegisteredViews()
	}
	return NewTemplate(options)
}

// Compile-time assertion that viewsHolder satisfies fiber.Views.
var _ fiber.Views = (*viewsHolder)(nil)

// viewsHolder is a lazily-bound fiber.Views. fiber v3 reads Config.Views from
// its internal config copy at render time and app.Config() returns a copy, so
// the value bound at fiber.New() time cannot be swapped afterwards. This
// indirection lets the default template set be compiled later — after all
// providers have booted — and handed to the already-created fiber app (see
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

// Load is a no-op until the default views have been compiled.
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

func walkTmplFiles(dir string, leftDelim string, fn func(filePath string, name string)) error {
	return filepath.WalkDir(dir, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) != ".tmpl" {
			return nil
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		name := extractDefineName(string(content), leftDelim)
		if name != "" {
			fn(filePath, name)
		}
		return nil
	})
}

func extractDefineName(content string, leftDelim string) string {
	if leftDelim == "" || leftDelim == "{{" {
		matches := defineRe.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1]
		}
		return ""
	}
	re, ok := customDefineRes.Load(leftDelim)
	if !ok {
		compiled := regexp.MustCompile(regexp.QuoteMeta(leftDelim) + `\s*define\s+"([^"]+)"`)
		customDefineRes.Store(leftDelim, compiled)
		re = compiled
	}
	matches := re.(*regexp.Regexp).FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
