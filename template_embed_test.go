package fiber

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
	mocksfoundation "github.com/goravel/framework/mocks/foundation"
	mockslog "github.com/goravel/framework/mocks/log"
	mocksview "github.com/goravel/framework/mocks/view"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/views
var embeddedViews embed.FS

// subFS roots an fs.FS at root, the shape View.LoadViewsFromFS() stores after
// its own validation (it additionally rejects a root that is missing or is not a
// directory, which subFS does not replicate).
func subFS(t *testing.T, fsys fs.FS, root string) fs.FS {
	t.Helper()
	sub, err := fs.Sub(fsys, root)
	require.NoError(t, err)
	return sub
}

// failingFS serves base except for failPath, where every open fails. It models a
// filesystem that turns unreadable after the root has been stat'ed: failPath may
// be a directory (the walk fails to list it) or a file (the walk fails to read it).
type failingFS struct {
	base     fs.FS
	failPath string
}

func (f failingFS) Open(name string) (fs.File, error) {
	if name == f.failPath {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.New("permission denied")}
	}
	return f.base.Open(name)
}

func renderView(t *testing.T, views fiber.Views, name string, data any) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, views.Render(&buf, name, data))
	return buf.String()
}

func renderOptions(t *testing.T, options RenderOptions, name string, data any) string {
	t.Helper()
	mv, err := NewTemplate(options)
	require.NoError(t, err)
	return renderView(t, mv, name, data)
}

func TestTemplate_EmbeddedViews(t *testing.T) {
	pkg := subFS(t, embeddedViews, "testdata/views")

	// Earlier tests may leave a spent log mock behind; every subtest that expects
	// a warning installs its own.
	previousLog := LogFacade
	LogFacade = nil
	defer func() { LogFacade = previousLog }()

	t.Run("embedded package views", func(t *testing.T) {
		assert.Equal(t, "Embedded Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{pkg}}, "page.tmpl", nil))
	})

	t.Run("nested layout, partial and block", func(t *testing.T) {
		assert.Equal(t,
			"<html><body><nav>Menu</nav><main><h1>Home</h1></main></body></html>",
			renderOptions(t, RenderOptions{ExtraFS: []fs.FS{pkg}}, "pages/home.tmpl", map[string]any{"Title": "Home", "Nav": "Menu"}),
		)
	})

	t.Run("app overrides embedded package", func(t *testing.T) {
		require.NoError(t, file.PutContent(path.Resource("views", "page.tmpl"), `{{ define "page.tmpl" }}App Content{{ end }}`))
		defer func() {
			assert.Nil(t, file.Remove(path.Resource("views")))
		}()

		assert.Equal(t, "App Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{pkg}}, "page.tmpl", nil))
	})

	t.Run("embedded package fallback for templates missing from app", func(t *testing.T) {
		require.NoError(t, file.PutContent(path.Resource("views", "other.tmpl"), `{{ define "other.tmpl" }}Other{{ end }}`))
		defer func() {
			assert.Nil(t, file.Remove(path.Resource("views")))
		}()

		mv, err := NewTemplate(RenderOptions{ExtraFS: []fs.FS{pkg}})
		require.NoError(t, err)

		assert.Equal(t, "Other", renderView(t, mv, "other.tmpl", nil))
		assert.Equal(t, "Embedded Content", renderView(t, mv, "page.tmpl", nil))
	})

	t.Run("directory package overrides embedded package", func(t *testing.T) {
		pkgDir, err := os.MkdirTemp("", "goravel-fiber-embed-dir-*")
		require.NoError(t, err)
		defer func() {
			assert.Nil(t, os.RemoveAll(pkgDir))
		}()
		require.NoError(t, file.PutContent(filepath.Join(pkgDir, "page.tmpl"), `{{ define "page.tmpl" }}Dir Content{{ end }}`))

		mockLog := mockslog.NewLog(t)
		LogFacade = mockLog
		defer func() { LogFacade = nil }()
		mockLog.EXPECT().Warningf("view collision: %q defined in %q and %q, using first", "page.tmpl", filepath.Join(pkgDir, "page.tmpl"), "fs[0]/page.tmpl").Return().Once()

		options := RenderOptions{ExtraPaths: []string{pkgDir}, ExtraFS: []fs.FS{pkg}}
		assert.Equal(t, "Dir Content", renderOptions(t, options, "page.tmpl", nil))
	})

	t.Run("multiple embedded packages", func(t *testing.T) {
		pkgB := fstest.MapFS{
			"views/bar.tmpl":     {Data: []byte(`{{ define "bar.tmpl" }}Bar{{ end }}`)},
			"views/sub/baz.tmpl": {Data: []byte(`{{ define "sub/baz.tmpl" }}Baz{{ end }}`)},
			"other/nope.tmpl":    {Data: []byte(`{{ define "nope.tmpl" }}Outside root{{ end }}`)},
		}

		mv, err := NewTemplate(RenderOptions{ExtraFS: []fs.FS{pkg, subFS(t, pkgB, "views")}})
		require.NoError(t, err)

		for name, expected := range map[string]string{
			"page.tmpl":    "Embedded Content",
			"bar.tmpl":     "Bar",
			"sub/baz.tmpl": "Baz",
		} {
			assert.Equal(t, expected, renderView(t, mv, name, nil), name)
		}

		assert.Nil(t, mv.engine.Lookup("nope.tmpl"), "files outside the registered root must not be loaded")
		assert.Nil(t, mv.engine.Lookup("missing.tmpl"))
	})

	t.Run("collision between embedded packages uses first", func(t *testing.T) {
		second := fstest.MapFS{
			"page.tmpl": {Data: []byte(`{{ define "page.tmpl" }}Second{{ end }}`)},
		}

		mockLog := mockslog.NewLog(t)
		LogFacade = mockLog
		defer func() { LogFacade = nil }()
		mockLog.EXPECT().Warningf("view collision: %q defined in %q and %q, using first", "page.tmpl", "fs[0]/page.tmpl", "fs[1]/page.tmpl").Return().Once()

		assert.Equal(t, "Embedded Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{pkg, second}}, "page.tmpl", nil))
	})

	t.Run("custom delims", func(t *testing.T) {
		custom := fstest.MapFS{
			"delim.tmpl": {Data: []byte(`{[ define "delim.tmpl" ]}Custom{[ end ]}`)},
		}

		options := RenderOptions{Delims: &Delims{Left: "{[", Right: "]}"}, ExtraFS: []fs.FS{custom}}
		assert.Equal(t, "Custom", renderOptions(t, options, "delim.tmpl", nil))
	})

	t.Run("unreadable root is skipped and warned about", func(t *testing.T) {
		missing, err := fs.Sub(fstest.MapFS{}, "does/not/exist")
		require.NoError(t, err)

		mockLog := mockslog.NewLog(t)
		LogFacade = mockLog
		defer func() { LogFacade = nil }()
		mockLog.EXPECT().Warningf("view source fs[%d] is unreadable, skipping: %v", 0, mock.Anything).Return().Once()

		assert.Equal(t, "Embedded Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{missing, pkg}}, "page.tmpl", nil))
	})

	t.Run("nil filesystem is skipped and warned about", func(t *testing.T) {
		mockLog := mockslog.NewLog(t)
		LogFacade = mockLog
		defer func() { LogFacade = nil }()
		mockLog.EXPECT().Warningf("view source fs[%d] is nil, skipping", 0).Return().Once()

		assert.Equal(t, "Embedded Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{nil, pkg}}, "page.tmpl", nil))
	})

	t.Run("skipped filesystem keeps its registration index in warnings", func(t *testing.T) {
		second := fstest.MapFS{
			"page.tmpl": {Data: []byte(`{{ define "page.tmpl" }}Second{{ end }}`)},
		}

		mockLog := mockslog.NewLog(t)
		LogFacade = mockLog
		defer func() { LogFacade = nil }()
		mockLog.EXPECT().Warningf("view source fs[%d] is nil, skipping", 0).Return().Once()
		mockLog.EXPECT().Warningf("view collision: %q defined in %q and %q, using first", "page.tmpl", "fs[1]/page.tmpl", "fs[2]/page.tmpl").Return().Once()

		assert.Equal(t, "Embedded Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{nil, pkg, second}}, "page.tmpl", nil))
	})

	t.Run("nil or unreadable filesystem without a log facade", func(t *testing.T) {
		missing, err := fs.Sub(fstest.MapFS{}, "does/not/exist")
		require.NoError(t, err)

		assert.Equal(t, "Embedded Content", renderOptions(t, RenderOptions{ExtraFS: []fs.FS{nil, missing, pkg}}, "page.tmpl", nil))
	})

	t.Run("only empty filesystems yields a template that cannot render", func(t *testing.T) {
		// A readable root holding no files: the source passes the stat guard and
		// is dropped because loadSource() finds nothing to parse.
		empty := fstest.MapFS{"sub": {Mode: fs.ModeDir}}

		mv, err := NewTemplate(RenderOptions{ExtraFS: []fs.FS{empty}})
		require.NoError(t, err)
		require.NotNil(t, mv)

		var buf bytes.Buffer
		assert.ErrorIs(t, mv.Render(&buf, "page.tmpl", nil), fiber.ErrInternalServerError)
	})

	t.Run("non-template and define-less files in an embedded filesystem are ignored", func(t *testing.T) {
		// //go:embed of a views directory picks up whatever else lives there. Only
		// .tmpl files with a define block are parsed, so nothing else can break
		// the compile or leak into the template set.
		assets := fstest.MapFS{
			"README.md":       {Data: []byte("# Package views {{ .Unclosed\n")},
			"assets/site.css": {Data: []byte("body { margin: 0 }")},
			"raw.tmpl":        {Data: []byte(`Raw {{ .Unclosed`)},
			"widget.tmpl":     {Data: []byte(`{{ define "widget.tmpl" }}Widget{{ end }}`)},
		}

		mv, err := NewTemplate(RenderOptions{ExtraFS: []fs.FS{assets}})
		require.NoError(t, err)

		assert.Equal(t, "Widget", renderView(t, mv, "widget.tmpl", nil))
		assert.Nil(t, mv.engine.Lookup("README.md"))
		assert.Nil(t, mv.engine.Lookup("site.css"))
		assert.Nil(t, mv.engine.Lookup("raw.tmpl"))
	})

	t.Run("names containing glob metacharacters are loaded literally", func(t *testing.T) {
		// Regression test: template names are resolved literally. Resolving them
		// as globs turns "page[1].tmpl" into the pattern "page1.tmpl", which either
		// loads the wrong file or fails the whole compile.
		pkgDir, err := os.MkdirTemp("", "goravel-fiber-embed-glob-*")
		require.NoError(t, err)
		defer func() {
			assert.Nil(t, os.RemoveAll(pkgDir))
		}()
		require.NoError(t, file.PutContent(filepath.Join(pkgDir, "dir[1].tmpl"), `{{ define "dir[1].tmpl" }}Dir Bracket{{ end }}`))

		embedded := fstest.MapFS{
			"page[1].tmpl": {Data: []byte(`{{ define "page[1].tmpl" }}Bracket{{ end }}`)},
			"page1.tmpl":   {Data: []byte(`{{ define "page1.tmpl" }}Decoy{{ end }}`)},
			"draft?.tmpl":  {Data: []byte(`{{ define "draft?.tmpl" }}Question{{ end }}`)},
		}

		mv, err := NewTemplate(RenderOptions{ExtraPaths: []string{pkgDir}, ExtraFS: []fs.FS{embedded}})
		require.NoError(t, err)

		for name, expected := range map[string]string{
			"dir[1].tmpl":  "Dir Bracket",
			"page[1].tmpl": "Bracket",
			"page1.tmpl":   "Decoy",
			"draft?.tmpl":  "Question",
		} {
			assert.Equal(t, expected, renderView(t, mv, name, nil), name)
		}
	})

	t.Run("regular file as a view path is skipped", func(t *testing.T) {
		// LoadViewsFrom() does not validate what it is given, and nothing stops
		// resources/views from being a file. Walking a file through os.DirFS
		// fails with "not a directory", which must not take the compile down.
		pkgDir, err := os.MkdirTemp("", "goravel-fiber-embed-file-*")
		require.NoError(t, err)
		defer func() {
			assert.Nil(t, os.RemoveAll(pkgDir))
		}()
		single := filepath.Join(pkgDir, "single.tmpl")
		require.NoError(t, file.PutContent(single, `{{ define "single.tmpl" }}Single{{ end }}`))

		appViews := path.Resource("views")
		require.NoError(t, file.PutContent(appViews, `{{ define "page.tmpl" }}App File{{ end }}`))
		defer func() {
			assert.Nil(t, file.Remove(appViews))
		}()

		mv, err := NewTemplate(RenderOptions{ExtraPaths: []string{single}, ExtraFS: []fs.FS{pkg}})
		require.NoError(t, err)

		assert.Equal(t, "Embedded Content", renderView(t, mv, "page.tmpl", nil))
		assert.Nil(t, mv.engine.Lookup("single.tmpl"))
	})

	t.Run("walk errors are propagated with the source path", func(t *testing.T) {
		base := fstest.MapFS{
			"page.tmpl":     {Data: []byte(`{{ define "page.tmpl" }}Page{{ end }}`)},
			"sub/deep.tmpl": {Data: []byte(`{{ define "sub/deep.tmpl" }}Deep{{ end }}`)},
		}

		for failPath, expected := range map[string]string{
			"sub":       "failed to walk view source fs[0]/sub: ",
			"page.tmpl": "failed to read view fs[0]/page.tmpl: ",
		} {
			t.Run(failPath, func(t *testing.T) {
				mv, err := NewTemplate(RenderOptions{ExtraFS: []fs.FS{failingFS{base: base, failPath: failPath}}})
				assert.ErrorContains(t, err, expected)
				assert.ErrorContains(t, err, "permission denied")
				assert.Nil(t, mv)
			})
		}
	})

	t.Run("invalid template returns parse error with the source path", func(t *testing.T) {
		broken := fstest.MapFS{
			"broken.tmpl": {Data: []byte(`{{ define "broken.tmpl" }}{{ .Unclosed`)},
		}

		mv, err := NewTemplate(RenderOptions{ExtraFS: []fs.FS{broken}})
		assert.ErrorContains(t, err, "failed to parse view fs[0]/broken.tmpl: ")
		assert.Nil(t, mv)
	})
}

func TestDefaultTemplate_EmbeddedViews(t *testing.T) {
	pkg := subFS(t, embeddedViews, "testdata/views")

	previousLog := LogFacade
	LogFacade = nil
	defer func() { LogFacade = previousLog }()

	t.Run("filesystems registered on the view facade", func(t *testing.T) {
		defer func() { ViewFacade = nil }()

		mockView := mocksview.NewView(t)
		mockView.EXPECT().RegisteredViews().Return(nil).Once()
		mockView.EXPECT().RegisteredViewFS().Return([]fs.FS{pkg}).Once()
		ViewFacade = mockView

		views, err := DefaultTemplate()
		require.NoError(t, err)
		assert.Equal(t, "Embedded Content", renderView(t, views, "page.tmpl", nil))
	})

	t.Run("directories rank above filesystems", func(t *testing.T) {
		defer func() { ViewFacade = nil }()

		pkgDir, err := os.MkdirTemp("", "goravel-fiber-embed-default-*")
		require.NoError(t, err)
		defer func() {
			assert.Nil(t, os.RemoveAll(pkgDir))
		}()
		require.NoError(t, file.PutContent(filepath.Join(pkgDir, "page.tmpl"), `{{ define "page.tmpl" }}Dir Content{{ end }}`))

		mockView := mocksview.NewView(t)
		mockView.EXPECT().RegisteredViews().Return([]string{pkgDir}).Once()
		mockView.EXPECT().RegisteredViewFS().Return([]fs.FS{pkg}).Once()
		ViewFacade = mockView

		views, err := DefaultTemplate()
		require.NoError(t, err)
		assert.Equal(t, "Dir Content", renderView(t, views, "page.tmpl", nil))
		assert.Equal(t, "<html><body><nav>Menu</nav><main><h1>Home</h1></main></body></html>",
			renderView(t, views, "pages/home.tmpl", map[string]any{"Title": "Home", "Nav": "Menu"}))
	})

	t.Run("view facade resolved from the container", func(t *testing.T) {
		// Defensive path: ViewFacade was never assigned, so DefaultTemplate()
		// resolves the facade through the application container.
		defer func() {
			App = nil
			ViewFacade = nil
		}()

		mockView := mocksview.NewView(t)
		mockView.EXPECT().RegisteredViews().Return(nil).Once()
		mockView.EXPECT().RegisteredViewFS().Return([]fs.FS{pkg}).Once()

		mockApp := mocksfoundation.NewApplication(t)
		mockApp.EXPECT().MakeView().Return(mockView).Once()
		App = mockApp
		ViewFacade = nil

		views, err := DefaultTemplate()
		require.NoError(t, err)
		assert.Equal(t, "Embedded Content", renderView(t, views, "page.tmpl", nil))
	})
}
