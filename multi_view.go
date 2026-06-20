package fiber

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/goravel/framework/support/file"
	"github.com/goravel/framework/support/path"
)

var defineRe = regexp.MustCompile(`\{\{\s*define\s+"([^"]+)"`)

type MultiView struct {
	mu         sync.RWMutex
	engine     *template.Template
	extraPaths []string
}

func NewMultiView(extraPaths []string) *MultiView {
	return &MultiView{
		extraPaths: extraPaths,
	}
}

func (m *MultiView) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.engine != nil {
		return nil
	}

	var files []string
	appDefines := make(map[string]string)
	pkgDefines := make(map[string]string)

	appDir := path.Resource("views")
	if file.Exists(appDir) {
		if err := walkTmplFiles(appDir, func(filePath string, name string) {
			appDefines[name] = filePath
			files = append(files, filePath)
		}); err != nil {
			return err
		}
	}

	for _, dir := range m.extraPaths {
		if !file.Exists(dir) {
			continue
		}
		if err := walkTmplFiles(dir, func(filePath string, name string) {
			if _, ok := appDefines[name]; ok {
				return
			}
			if existing, ok := pkgDefines[name]; ok {
				LogFacade.Warningf("view collision: %q defined in %q and %q, using first", name, existing, filePath)
				return
			}
			pkgDefines[name] = filePath
			files = append(files, filePath)
		}); err != nil {
			return err
		}
	}

	if len(files) == 0 {
		return nil
	}

	tmpl, err := template.New("").ParseFiles(files...)
	if err != nil {
		return err
	}
	m.engine = tmpl
	return nil
}

func (m *MultiView) Render(w io.Writer, name string, data interface{}, layouts ...string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.engine == nil {
		return fiber.ErrInternalServerError
	}

	return m.engine.ExecuteTemplate(w, name, data)
}

func walkTmplFiles(dir string, fn func(filePath string, name string)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".tmpl" {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		matches := defineRe.FindStringSubmatch(string(content))
		if len(matches) > 1 {
			fn(filePath, matches[1])
		}
	}
	return nil
}
