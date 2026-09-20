package fiber

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
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
	Delims      *Delims
	FuncMap     template.FuncMap
	ExtraPaths  []string
	ExtraViewFS []fs.FS
}

// Template implements fiber.Views for multi-directory template loading.
// Templates are loaded from the app's resources/views first, then from
// registered package directories, then from registered embedded filesystems.
// App-defined templates take priority; collisions between packages log a
// warning.
//
// Precedence is applied per file, keyed on that file's first {{ define }} block
// (see extractDefineName): a package file is only skipped when that first name
// is already claimed, so any additional {{ define }} blocks in the same file
// still register and may replace templates from higher-priority sources. This
// preserves the existing on-disk behavior.
type Template struct {
	mu     sync.RWMutex
	engine *template.Template
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
	fsys fs.FS
	tier viewTier
	// label identifies the source in warnings: the directory path for tierApp
	// and tierDir, "fs[i]" (i being the LoadViewsFromFS registration index) for
	// tierFS.
	label string
}

// pathOf renders name, a slash-separated path inside the source, for display.
func (s viewSource) pathOf(name string) string {
	if s.tier == tierFS {
		return s.label + "/" + name
	}
	return filepath.Join(s.label, filepath.FromSlash(name))
}

// viewDefines tracks the template name each precedence tier has already claimed,
// so the first source to define a name wins. Only a file's first {{ define }}
// block is claimed (see extractDefineName), so precedence is enforced per file,
// not per define block.
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

// isDir reports whether dir exists and is a directory. Non-directory paths
// (regular files, missing paths) are ignored: a real directory is required
// because wrapping a regular file in os.DirFS would make the walk fail. As a
// deliberate consequence, a direct .tmpl file passed as the app views path or an
// ExtraPaths entry is no longer parsed, whereas the previous filepath.WalkDir
// code parsed such a file; non-.tmpl regular files were already ignored.
func isDir(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// viewSources returns every existing template source in precedence order: the
// application's resources/views, then directories in options.ExtraPaths, then
// filesystems in options.ExtraViewFS, each in registration/argument order.
func viewSources(options RenderOptions) []viewSource {
	sources := make([]viewSource, 0, 1+len(options.ExtraPaths)+len(options.ExtraViewFS))

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
	for i, fsys := range options.ExtraViewFS {
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

// loadSource walks source and parses every .tmpl template it contributes into
// instance, reporting whether it contributed any. A file with no {{ define }}
// block is skipped, preserving the existing on-disk behavior.
func loadSource(instance *template.Template, source viewSource, leftDelim string, defines *viewDefines) (bool, error) {
	contributed := false

	err := fs.WalkDir(source.fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %w", source.pathOf(name), err)
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".tmpl" {
			return nil
		}

		content, err := fs.ReadFile(source.fsys, name)
		if err != nil {
			return fmt.Errorf("%s: %w", source.pathOf(name), err)
		}
		text := string(content)

		templateName := extractDefineName(text, leftDelim)
		if templateName == "" {
			return nil
		}
		if !defines.claim(source, name, templateName) {
			return nil
		}

		// Mirror ParseFiles: associate the file under its base name while any
		// {{ define }} blocks register their own names. Content is parsed directly
		// instead of via ParseFS, which resolves names through fs.Glob and would
		// mangle or fail on names containing "[", "*" or "?".
		if _, err := instance.New(d.Name()).Parse(text); err != nil {
			return fmt.Errorf("%s: %w", source.pathOf(name), err)
		}
		contributed = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return contributed, nil
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

// DefaultTemplate creates a Template from the app views plus the package view
// directories returned by ViewFacade.RegisteredViews() and the package view
// filesystems returned by ViewFacade.RegisteredViewFS(). Returns a valid
// fiber.Views even when there are no template files.
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
		options.ExtraViewFS = viewFacade.RegisteredViewFS()
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
