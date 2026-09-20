package fiber

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
	"github.com/goravel/framework/contracts/log"
	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/views
var embeddedViews embed.FS

// subFS returns fsys rooted at root. LoadViewsFromFS additionally rejects a
// missing or non-directory root; this helper assumes root exists.
func subFS(t *testing.T, fsys fs.FS, root string) fs.FS {
	t.Helper()
	sub, err := fs.Sub(fsys, root)
	require.NoError(t, err)
	return sub
}

// failingFS delegates to base but fails Open for failPath, simulating an
// unreadable root (failPath ".") or a walk error (a path inside the tree).
type failingFS struct {
	base     fs.FS
	failPath string
}

func (f failingFS) Open(name string) (fs.File, error) {
	if name == f.failPath {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return f.base.Open(name)
}

// renderTemplate compiles options and renders name, requiring no error.
func renderTemplate(t *testing.T, options RenderOptions, name string, data any) string {
	t.Helper()
	tmpl, err := NewTemplate(options)
	require.NoError(t, err)
	require.NotNil(t, tmpl)

	var buf bytes.Buffer
	require.NoError(t, tmpl.Render(&buf, name, data))
	return buf.String()
}

// writeAppView writes an app view and schedules removal of the whole app views
// directory so subtests do not leak state into each other.
func writeAppView(t *testing.T, name, content string) {
	t.Helper()
	require.NoError(t, file.PutContent(path.Resource("views", name), content))
	t.Cleanup(func() {
		require.NoError(t, file.Remove(path.Resource("views")))
	})
}

// useLogFacade installs facade as the global LogFacade for the duration of the
// calling subtest and restores the previous value when it ends, so subtests do
// not depend on each other's mutations of the package global.
func useLogFacade(t *testing.T, facade log.Log) {
	t.Helper()
	previous := LogFacade
	LogFacade = facade
	t.Cleanup(func() {
		LogFacade = previous
	})
}

func TestTemplate_EmbeddedViews(t *testing.T) {
	previousLogFacade := LogFacade
	t.Cleanup(func() {
		LogFacade = previousLogFacade
	})

	viewFS := func() fs.FS { return subFS(t, embeddedViews, "testdata/views") }

	t.Run("embedded package views", func(t *testing.T) {
		got := renderTemplate(t, RenderOptions{ExtraViewFS: []fs.FS{viewFS()}}, "page.tmpl", nil)
		assert.Equal(t, "Embedded Content", got)
	})

	t.Run("regular file at resources/views is ignored", func(t *testing.T) {
		require.NoError(t, file.PutContent(path.Resource("views"), `{{ define "app-file.tmpl" }}Not Parsed{{ end }}`))
		t.Cleanup(func() {
			require.NoError(t, file.Remove(path.Resource("views")))
		})

		options := RenderOptions{ExtraViewFS: []fs.FS{viewFS()}}
		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))

		// A regular file at the app views path is ignored, not parsed.
		tmpl, err := NewTemplate(options)
		require.NoError(t, err)
		var buf bytes.Buffer
		assert.Error(t, tmpl.Render(&buf, "app-file.tmpl", nil))
	})

	t.Run("regular file in ExtraPaths is ignored", func(t *testing.T) {
		regularFile := filepath.Join(t.TempDir(), "not-a-dir.txt")
		require.NoError(t, file.PutContent(regularFile, "not a directory"))

		options := RenderOptions{ExtraPaths: []string{regularFile}, ExtraViewFS: []fs.FS{viewFS()}}
		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))
	})

	t.Run("regular .tmpl file in ExtraPaths is ignored", func(t *testing.T) {
		// The previous filepath.WalkDir implementation parsed a regular .tmpl
		// file passed directly as an ExtraPaths entry; the directory gate now
		// ignores it. Pin that deliberate behavior change.
		regularFile := filepath.Join(t.TempDir(), "standalone.tmpl")
		require.NoError(t, file.PutContent(regularFile, `{{ define "standalone.tmpl" }}Not Parsed{{ end }}`))

		options := RenderOptions{ExtraPaths: []string{regularFile}, ExtraViewFS: []fs.FS{viewFS()}}
		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))

		tmpl, err := NewTemplate(options)
		require.NoError(t, err)
		var buf bytes.Buffer
		assert.Error(t, tmpl.Render(&buf, "standalone.tmpl", nil))
	})

	t.Run("nested layout, partial and block", func(t *testing.T) {
		got := renderTemplate(t, RenderOptions{ExtraViewFS: []fs.FS{viewFS()}}, "pages/home.tmpl", map[string]any{
			"Title": "Home",
			"Nav":   "Menu",
		})
		assert.Equal(t, "<html><body><nav>Menu</nav><main><h1>Home</h1></main></body></html>", got)
	})

	t.Run("app view overrides embedded", func(t *testing.T) {
		writeAppView(t, "page.tmpl", `{{ define "page.tmpl" }}App Content{{ end }}`)

		got := renderTemplate(t, RenderOptions{ExtraViewFS: []fs.FS{viewFS()}}, "page.tmpl", nil)
		assert.Equal(t, "App Content", got)
	})

	t.Run("embedded fallback when app only defines some names", func(t *testing.T) {
		writeAppView(t, "page.tmpl", `{{ define "page.tmpl" }}App Content{{ end }}`)

		fsys := fstest.MapFS{
			"page.tmpl":  {Data: []byte(`{{ define "page.tmpl" }}Embedded Page{{ end }}`)},
			"other.tmpl": {Data: []byte(`{{ define "other.tmpl" }}Embedded Other{{ end }}`)},
		}
		options := RenderOptions{ExtraViewFS: []fs.FS{fsys}}

		assert.Equal(t, "App Content", renderTemplate(t, options, "page.tmpl", nil))
		assert.Equal(t, "Embedded Other", renderTemplate(t, options, "other.tmpl", nil))
	})

	t.Run("directory package overrides embedded and warns", func(t *testing.T) {
		pkgDir := t.TempDir()
		require.NoError(t, file.PutContent(filepath.Join(pkgDir, "page.tmpl"), `{{ define "page.tmpl" }}Dir Content{{ end }}`))

		mockLog := mockslog.NewLog(t)
		useLogFacade(t, mockLog)
		mockLog.EXPECT().Warningf(
			"view collision: %q defined in %q and %q, using first",
			"page.tmpl", filepath.Join(pkgDir, "page.tmpl"), "fs[0]/page.tmpl",
		).Once()

		options := RenderOptions{
			ExtraPaths:  []string{pkgDir},
			ExtraViewFS: []fs.FS{viewFS()},
		}
		assert.Equal(t, "Dir Content", renderTemplate(t, options, "page.tmpl", nil))
	})

	t.Run("nested package directory renders", func(t *testing.T) {
		pkgDir := t.TempDir()
		require.NoError(t, file.PutContent(filepath.Join(pkgDir, "sub", "x.tmpl"), `{{ define "sub/x.tmpl" }}Nested Package{{ end }}`))

		options := RenderOptions{ExtraPaths: []string{pkgDir}}
		assert.Equal(t, "Nested Package", renderTemplate(t, options, "sub/x.tmpl", nil))
	})

	t.Run("multiple embedded packages and subdirectories", func(t *testing.T) {
		pkg1 := fstest.MapFS{
			"views/a.tmpl":     {Data: []byte(`{{ define "a.tmpl" }}A{{ end }}`)},
			"views/sub/b.tmpl": {Data: []byte(`{{ define "sub/b.tmpl" }}B{{ end }}`)},
			"outside.tmpl":     {Data: []byte(`{{ define "outside.tmpl" }}Outside{{ end }}`)},
		}
		pkg2 := fstest.MapFS{
			"c.tmpl": {Data: []byte(`{{ define "c.tmpl" }}C{{ end }}`)},
		}
		options := RenderOptions{ExtraViewFS: []fs.FS{subFS(t, pkg1, "views"), pkg2}}

		assert.Equal(t, "A", renderTemplate(t, options, "a.tmpl", nil))
		assert.Equal(t, "B", renderTemplate(t, options, "sub/b.tmpl", nil))
		assert.Equal(t, "C", renderTemplate(t, options, "c.tmpl", nil))

		tmpl, err := NewTemplate(options)
		require.NoError(t, err)
		var buf bytes.Buffer
		assert.Error(t, tmpl.Render(&buf, "outside.tmpl", nil))
	})

	t.Run("collision between embedded packages keeps the first and warns", func(t *testing.T) {
		first := fstest.MapFS{
			"page.tmpl": {Data: []byte(`{{ define "page.tmpl" }}First{{ end }}`)},
		}
		second := fstest.MapFS{
			"page.tmpl": {Data: []byte(`{{ define "page.tmpl" }}Second{{ end }}`)},
		}

		mockLog := mockslog.NewLog(t)
		useLogFacade(t, mockLog)
		mockLog.EXPECT().Warningf(
			"view collision: %q defined in %q and %q, using first",
			"page.tmpl", "fs[0]/page.tmpl", "fs[1]/page.tmpl",
		).Once()

		options := RenderOptions{ExtraViewFS: []fs.FS{first, second}}
		assert.Equal(t, "First", renderTemplate(t, options, "page.tmpl", nil))
	})

	t.Run("custom delimiters", func(t *testing.T) {
		fsys := fstest.MapFS{
			"custom.tmpl": {Data: []byte(`{[ define "custom.tmpl" ]}Custom Embedded{[ end ]}`)},
		}
		options := RenderOptions{
			Delims:      &Delims{Left: "{[", Right: "]}"},
			ExtraViewFS: []fs.FS{fsys},
		}
		assert.Equal(t, "Custom Embedded", renderTemplate(t, options, "custom.tmpl", nil))
	})

	t.Run("unreadable root is skipped and warned", func(t *testing.T) {
		mockLog := mockslog.NewLog(t)
		useLogFacade(t, mockLog)
		mockLog.EXPECT().Warningf("view source fs[%d] is unreadable, skipping: %v", 0, mock.Anything).Once()

		options := RenderOptions{ExtraViewFS: []fs.FS{failingFS{base: embeddedViews, failPath: "."}, viewFS()}}
		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))
	})

	t.Run("nil filesystem is skipped and warned", func(t *testing.T) {
		mockLog := mockslog.NewLog(t)
		useLogFacade(t, mockLog)
		mockLog.EXPECT().Warningf("view source fs[%d] is nil, skipping", 0).Once()

		options := RenderOptions{ExtraViewFS: []fs.FS{nil, viewFS()}}
		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))
	})

	t.Run("nil and unreadable sources do not panic without a log facade", func(t *testing.T) {
		useLogFacade(t, nil)

		options := RenderOptions{ExtraViewFS: []fs.FS{nil, failingFS{base: embeddedViews, failPath: "."}, viewFS()}}
		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))
	})

	t.Run("glob metacharacters in names are loaded literally", func(t *testing.T) {
		fsys := fstest.MapFS{
			"dir[1].tmpl":  {Data: []byte(`{{ define "dir[1].tmpl" }}Dir Bracket{{ end }}`)},
			"page[1].tmpl": {Data: []byte(`{{ define "page[1].tmpl" }}Page Bracket{{ end }}`)},
			"page1.tmpl":   {Data: []byte(`{{ define "page1.tmpl" }}Page One{{ end }}`)},
			"draft?.tmpl":  {Data: []byte(`{{ define "draft?.tmpl" }}Draft{{ end }}`)},
		}
		options := RenderOptions{ExtraViewFS: []fs.FS{fsys}}

		assert.Equal(t, "Dir Bracket", renderTemplate(t, options, "dir[1].tmpl", nil))
		assert.Equal(t, "Page Bracket", renderTemplate(t, options, "page[1].tmpl", nil))
		assert.Equal(t, "Page One", renderTemplate(t, options, "page1.tmpl", nil))
		assert.Equal(t, "Draft", renderTemplate(t, options, "draft?.tmpl", nil))
	})

	t.Run("walk errors are propagated", func(t *testing.T) {
		tmpl, err := NewTemplate(RenderOptions{
			ExtraViewFS: []fs.FS{failingFS{base: embeddedViews, failPath: "testdata/views/page.tmpl"}},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "permission denied")
		assert.Nil(t, tmpl)
	})

	t.Run("invalid template returns a parse error", func(t *testing.T) {
		fsys := fstest.MapFS{
			"broken.tmpl": {Data: []byte(`{{ define "broken.tmpl" }}{{ .Unclosed`)},
		}
		tmpl, err := NewTemplate(RenderOptions{ExtraViewFS: []fs.FS{fsys}})
		require.Error(t, err)
		assert.Nil(t, tmpl)
	})

	t.Run("empty filesystems yield a valid empty template", func(t *testing.T) {
		tmpl, err := NewTemplate(RenderOptions{ExtraViewFS: []fs.FS{os.DirFS(t.TempDir())}})
		require.NoError(t, err)
		require.NotNil(t, tmpl)

		var buf bytes.Buffer
		err = tmpl.Render(&buf, "anything.tmpl", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, fiber.ErrInternalServerError)
	})

	t.Run("non-template files and tmpl files without define are ignored", func(t *testing.T) {
		fsys := fstest.MapFS{
			"README.md":       {Data: []byte("# readme")},
			"assets/site.css": {Data: []byte("body{}")},
			"plain.tmpl":      {Data: []byte("no define block here")},
			"page.tmpl":       {Data: []byte(`{{ define "page.tmpl" }}Embedded Content{{ end }}`)},
		}
		options := RenderOptions{ExtraViewFS: []fs.FS{fsys}}

		assert.Equal(t, "Embedded Content", renderTemplate(t, options, "page.tmpl", nil))

		tmpl, err := NewTemplate(options)
		require.NoError(t, err)
		var buf bytes.Buffer
		assert.Error(t, tmpl.Render(&buf, "plain.tmpl", nil))
		assert.Error(t, tmpl.Render(&buf, "README.md", nil))
		assert.Error(t, tmpl.Render(&buf, "assets/site.css", nil))
	})
}
