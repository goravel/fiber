package fiber

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	stdpath "path"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/gofiber/fiber/v3"
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
	ExtraFS    []fs.FS
}

// viewTier is the precedence class of a view source. Lower-numbered tiers take
// precedence, in the order tierApp, tierDir, tierFS.
type viewTier int

const (
	tierApp viewTier = iota
	tierDir
	tierFS
)

// viewSource is a filesystem that contributes templates at a given precedence tier.
type viewSource struct {
	fsys  fs.FS
	tier  viewTier
	label string
}

// pathOf renders name, a slash-separated path inside the source, for display.
func (s viewSource) pathOf(name string) string {
	if s.tier == tierFS {
		return stdpath.Join(s.label, name)
	}
	return filepath.Join(s.label, filepath.FromSlash(name))
}

// viewDefines tracks the template name each precedence tier has already claimed,
// so the first source to define a name wins.
type viewDefines struct {
	app map[string]string
	pkg map[string]string
}

// claim records templateName for source and reports whether source may contribute
// it. A name already claimed by the application is dropped silently; one already
// claimed by an earlier package source is dropped with a warning.
func (d *viewDefines) claim(source viewSource, name, templateName string) bool {
	fullPath := source.pathOf(name)

	if source.tier == tierApp {
		d.app[templateName] = fullPath
		return true
	}
	if _, ok := d.app[templateName]; ok {
		return false
	}
	if prevFile, ok := d.pkg[templateName]; ok {
		if LogFacade != nil {
			LogFacade.Warningf("view collision: %q defined in %q and %q, using first", templateName, prevFile, fullPath)
		}
		return false
	}

	d.pkg[templateName] = fullPath
	return true
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
// directory, any extra paths and any extra filesystems. Templates without a
// {{ define }} block are skipped. If no files are found, the Template is still
// valid but Render will return an error for any template name.
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

	defines := &viewDefines{app: make(map[string]string), pkg: make(map[string]string)}
	loaded := false

	for _, source := range viewSources(options) {
		contributed, err := loadSource(instance, source, leftDelim, defines)
		if err != nil {
			return nil, err
		}
		loaded = loaded || contributed
	}

	if !loaded {
		return &Template{}, nil
	}

	return &Template{engine: instance}, nil
}

// DefaultTemplate creates a Template with package view directories from
// ViewFacade.RegisteredViews() and package view filesystems from
// ViewFacade.RegisteredViewFS(). Returns a valid fiber.Views even when there
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
		options.ExtraFS = viewFacade.RegisteredViewFS()
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

// viewSources returns every existing template source in precedence order: the
// application's resources/views, then options.ExtraPaths (directories registered
// via LoadViewsFrom), then options.ExtraFS (filesystems registered via
// LoadViewsFromFS), each in registration order. A path that is missing or is
// not a directory is skipped: walking it through os.DirFS would fail the whole
// compile.
func viewSources(options RenderOptions) []viewSource {
	var sources []viewSource

	if dir := path.Resource("views"); isDir(dir) {
		sources = append(sources, viewSource{fsys: os.DirFS(dir), tier: tierApp, label: dir})
	}

	for _, dir := range options.ExtraPaths {
		if isDir(dir) {
			sources = append(sources, viewSource{fsys: os.DirFS(dir), tier: tierDir, label: dir})
		}
	}

	// Labels keep the registration index, so a skipped filesystem still consumes
	// its slot and warnings point at the position the package registered.
	for i, fsys := range options.ExtraFS {
		if fsys == nil {
			if LogFacade != nil {
				LogFacade.Warningf("view source fs[%d] is nil, skipping", i)
			}
			continue
		}
		if _, err := fs.Stat(fsys, "."); err != nil {
			if LogFacade != nil {
				LogFacade.Warningf("view source fs[%d] is unreadable, skipping: %v", i, err)
			}
			continue
		}
		sources = append(sources, viewSource{fsys: fsys, tier: tierFS, label: fmt.Sprintf("fs[%d]", i)})
	}

	return sources
}

func isDir(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// loadSource walks source and parses every .tmpl file it contributes into
// instance, reporting whether it contributed any. Each file is read once and
// parsed immediately, so only one file's content is held at a time. Files
// without a define block are skipped. Errors name the source they came from:
// os.DirFS reports paths relative to the directory, and an fs.FS has no path of
// its own.
func loadSource(instance *template.Template, source viewSource, leftDelim string, defines *viewDefines) (bool, error) {
	contributed := false

	err := fs.WalkDir(source.fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed to walk view source %s: %w", source.pathOf(name), err)
		}
		if d.IsDir() {
			return nil
		}
		if stdpath.Ext(name) != ".tmpl" {
			return nil
		}

		content, err := fs.ReadFile(source.fsys, name)
		if err != nil {
			return fmt.Errorf("failed to read view %s: %w", source.pathOf(name), err)
		}
		text := string(content)

		templateName := extractDefineName(text, leftDelim)
		if templateName == "" {
			return nil
		}
		if !defines.claim(source, name, templateName) {
			return nil
		}

		// Mirrors html/template.ParseFiles: every file becomes an associated
		// template named after its base. The content is parsed directly instead
		// of going through ParseFS, which would re-read the file and — because it
		// resolves names through fs.Glob — mangle or fail on any name containing
		// "[", "*" or "?".
		if _, err := instance.New(stdpath.Base(name)).Parse(text); err != nil {
			return fmt.Errorf("failed to parse view %s: %w", source.pathOf(name), err)
		}
		contributed = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return contributed, nil
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
