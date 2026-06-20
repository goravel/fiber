package fiber

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	mockslog "github.com/goravel/framework/mocks/log"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMultiView_AppViewsOnly(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "hello.tmpl"), `{{ define "hello" }}Hello World{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv := NewMultiView(nil)
	err := mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "hello", nil)
	require.Nil(t, err)
	assert.Equal(t, "Hello World", buf.String())
}

func TestMultiView_LoadIdempotent(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "idem.tmpl"), `{{ define "idem" }}Idempotent{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv := NewMultiView(nil)
	err := mv.Load()
	require.Nil(t, err)

	err = mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "idem", nil)
	require.Nil(t, err)
	assert.Equal(t, "Idempotent", buf.String())
}

func TestMultiView_PackageViews(t *testing.T) {
	pkgDir, err := os.MkdirTemp("", "goravel-fiber-pkgviews-*")
	require.Nil(t, err)
	defer func() {
		assert.Nil(t, os.RemoveAll(pkgDir))
	}()

	assert.Nil(t, file.PutContent(filepath.Join(pkgDir, "pkg.tmpl"), `{{ define "pkg" }}Package View{{ end }}`))

	mv := NewMultiView([]string{pkgDir})
	err = mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "pkg", nil)
	require.Nil(t, err)
	assert.Equal(t, "Package View", buf.String())
}

func TestMultiView_AppOverridesPackage(t *testing.T) {
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

	mv := NewMultiView([]string{pkgDir})
	err = mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "override", nil)
	require.Nil(t, err)
	assert.Equal(t, "App Version", buf.String())
}

func TestMultiView_PackageCollision(t *testing.T) {
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

	mockLog := &mockslog.Log{}
	LogFacade = mockLog
	mockLog.EXPECT().Warningf("view collision: %q defined in %q and %q, using first", "collide", mock.Anything, mock.Anything).Once()

	mv := NewMultiView([]string{pkgDir1, pkgDir2})
	err = mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "collide", nil)
	require.Nil(t, err)
	assert.Equal(t, "First", buf.String())

	mockLog.AssertExpectations(t)
}

func TestMultiView_NoTemplates(t *testing.T) {
	mv := NewMultiView(nil)
	err := mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "nonexistent", nil)
	assert.Error(t, err)
}

func TestMultiView_TemplateWithData(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "data.tmpl"), `{{ define "data" }}{{ .Name }} is {{ .Age }}{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv := NewMultiView(nil)
	err := mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "data", map[string]any{
		"Name": "Alice",
		"Age":  30,
	})
	require.Nil(t, err)
	assert.Equal(t, "Alice is 30", buf.String())
}

func TestMultiView_LoadErrorOnInvalidDir(t *testing.T) {
	mv := NewMultiView([]string{"/nonexistent/dir/that/should/not/exist"})
	err := mv.Load()
	require.Nil(t, err)
}

func TestMultiView_RenderNonexistentTemplate(t *testing.T) {
	assert.Nil(t, file.PutContent(path.Resource("views", "exists.tmpl"), `{{ define "exists" }}I exist{{ end }}`))
	defer func() {
		assert.Nil(t, file.Remove(path.Resource("views")))
	}()

	mv := NewMultiView(nil)
	err := mv.Load()
	require.Nil(t, err)

	var buf bytes.Buffer
	err = mv.Render(&buf, "nonexistent", nil)
	assert.Error(t, err)

	err = mv.Render(&buf, "exists", nil)
	require.Nil(t, err)
	assert.Equal(t, "I exist", buf.String())
}
