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

type Delims struct {
	Left  string
	Right string
}

type RenderOptions struct {
	Delims     *Delims
	FuncMap    template.FuncMap
	ExtraPaths []string
}

type Template struct {
	mu     sync.RWMutex
	engine *template.Template
}

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

func DefaultTemplate() (fiber.Views, error) {
	options := RenderOptions{}
	if ViewFacade != nil {
		options.ExtraPaths = ViewFacade.RegisteredViews()
	}
	return NewTemplate(options)
}

func (m *Template) Load() error {
	return nil
}

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
		name := extractDefineName(string(content), leftDelim)
		if name != "" {
			fn(filePath, name)
		}
	}
	return nil
}

func extractDefineName(content string, leftDelim string) string {
	if leftDelim == "" || leftDelim == "{{" {
		matches := defineRe.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1]
		}
		return ""
	}
	re := regexp.MustCompile(regexp.QuoteMeta(leftDelim) + `\s*define\s+"([^"]+)"`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
