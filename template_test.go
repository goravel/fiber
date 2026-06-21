package fiber

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"testing"

	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplate_AppViewsOnly(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "hello.tmpl"), `{{ define "hello" }}Hello World{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "hello", nil)
	require.Nil(t, err)
	assert.Equal(t, "Hello World", buf.String())
}

func TestTemplate_LoadIdempotent(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "idem.tmpl"), `{{ define "idem" }}Idempotent{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	err = mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "idem", nil)
	require.Nil(t, err)
	assert.Equal(t, "Idempotent", buf.String())
}

func TestTemplate_PackageViews(t *testing.T) {
	pkgDir, err := os.MkdirTemp("", "goravel-fiber-pkgviews-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir))
	}()

	assert.Nil(t, file.PutContent(filepath.Join(pkgDir, "pkg.tmpl"), `{{ define "pkg" }}Package View{{ end }}`))

	mv, err := NewTemplate(RenderOptions{ExtraPaths: []string{pkgDir}})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "pkg", nil)
	require.Nil(t, err)
	assert.Equal(t, "Package View", buf.String())
}

func TestTemplate_AppOverridesPackage(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "override.tmpl"), `{{ define "override" }}App Version{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	pkgDir, err := os.MkdirTemp("", "goravel-fiber-override-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir))
	}()

	assert.Nil(t, file.PutContent(filepath.Join(pkgDir, "override.tmpl"), `{{ define "override" }}Package Version{{ end }}`))

	mv, err := NewTemplate(RenderOptions{ExtraPaths: []string{pkgDir}})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "override", nil)
	require.Nil(t, err)
	assert.Equal(t, "App Version", buf.String())
}

func TestTemplate_PackageCollision(t *testing.T) {
	pkgDir1, err := os.MkdirTemp("", "goravel-fiber-collision1-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir1))
	}()

	pkgDir2, err := os.MkdirTemp("", "goravel-fiber-collision2-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir2))
	}()

	assert.Nil(t, file.PutContent(filepath.Join(pkgDir1, "collide.tmpl"), `{{ define "collide" }}First{{ end }}`))
	assert.Nil(t, file.PutContent(filepath.Join(pkgDir2, "collide.tmpl"), `{{ define "collide" }}Second{{ end }}`))

	collide1Path := filepath.Join(pkgDir1, "collide.tmpl")
	collide2Path := filepath.Join(pkgDir2, "collide.tmpl")

	mockLog := mockslog.NewLog(t)
	LogFacade = mockLog
	mockLog.EXPECT().Warningf("view collision: %q defined in %q and %q, using first", "collide", collide1Path, collide2Path).Once()

	mv, err := NewTemplate(RenderOptions{ExtraPaths: []string{pkgDir1, pkgDir2}})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "collide", nil)
	require.Nil(t, err)
	assert.Equal(t, "First", buf.String())
}

func TestTemplate_NoTemplates(t *testing.T) {
	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "nonexistent", nil)
	assert.Error(t, err)
}

func TestTemplate_TemplateWithData(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "data.tmpl"), `{{ define "data" }}{{ .Name }} is {{ .Age }}{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "data", map[string]any{
		"Name": "Alice",
		"Age":  30,
	})
	require.Nil(t, err)
	assert.Equal(t, "Alice is 30", buf.String())
}

func TestTemplate_NonexistentDirIgnored(t *testing.T) {
	mv, err := NewTemplate(RenderOptions{ExtraPaths: []string{"/nonexistent/dir/that/should/not/exist"}})
	require.Nil(t, err)
	require.NotNil(t, mv)
}

func TestTemplate_RenderNonexistentTemplate(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "exists.tmpl"), `{{ define "exists" }}I exist{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "nonexistent", nil)
	assert.Error(t, err)

	err = mv.Render(&buf, "exists", nil)
	require.Nil(t, err)
	assert.Equal(t, "I exist", buf.String())
}

func TestNewTemplate_CustomDelimiters(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "custom.tmpl"), `{[ define "custom" ]}Custom Delimiters{[ end ]}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{
		Delims: &Delims{Left: "{[", Right: "]}"},
	})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "custom", nil)
	require.Nil(t, err)
	assert.Equal(t, "Custom Delimiters", buf.String())
}

func TestNewTemplate_FuncMap(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "funcmap.tmpl"), `{{ define "funcmap" }}{{ upper . }}{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{
		FuncMap: template.FuncMap{
			"upper": func(s string) string { return s + "!" },
		},
	})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "funcmap", "hello")
	require.Nil(t, err)
	assert.Equal(t, "hello!", buf.String())
}

func TestTemplate_SubdirectoryViews(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "admin", "index.tmpl"), `{{ define "admin/index.tmpl" }}{{ .greeting }}, {{ .name }}{{ end }}`))
	assert.Nil(t, file.PutContent(path.Resource("views", "admin", "sidebar.tmpl"), `{{ define "admin/sidebar" }}{{ .greeting }}, {{ .name }}{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	data := map[string]any{"greeting": "Hello", "name": "Bowen"}

	var buf bytes.Buffer
	err = mv.Render(&buf, "admin/index.tmpl", data)
	require.Nil(t, err)
	assert.Equal(t, "Hello, Bowen", buf.String())

	buf.Reset()
	err = mv.Render(&buf, "admin/sidebar", data)
	require.Nil(t, err)
	assert.Equal(t, "Hello, Bowen", buf.String())
}

func TestNewTemplate_Layouts(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "layout.tmpl"), `{{ define "layout" }}<html>{{ template "content" . }}</html>{{ end }}`))
	assert.Nil(t, file.PutContent(path.Resource("views", "content.tmpl"), `{{ define "content" }}<p>{{ . }}</p>{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv, err := NewTemplate(RenderOptions{})
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "content", "Hello Layout", "layout")
	require.Nil(t, err)
	assert.Equal(t, "<html><p>Hello Layout</p></html>", buf.String())
}
