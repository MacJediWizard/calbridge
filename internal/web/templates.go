package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin/render"
)

//go:embed templates/*.html templates/sources/*.html templates/partials/*.html
var templatesFS embed.FS

// TemplateFuncs returns custom template functions.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"divide": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"plus": func(a, b int) int {
			return a + b
		},
		"minus": func(a, b int) int {
			return a - b
		},
		"multiply": func(a, b int) int {
			return a * b
		},
	}
}

// HTMLTemplates implements gin's render.HTMLRender interface with proper layout support.
type HTMLTemplates struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
}

// Instance returns a render.Render implementation for a specific template.
func (h *HTMLTemplates) Instance(name string, data interface{}) render.Render {
	h.mu.RLock()
	tmpl, ok := h.templates[name]
	h.mu.RUnlock()

	if !ok {
		// Return a simple error template
		return &templateRender{
			Template: template.Must(template.New("error").Parse("<html><body>Template not found: {{.}}</body></html>")),
			Data:     name,
		}
	}

	return &templateRender{
		Template: tmpl,
		Data:     data,
	}
}

type templateRender struct {
	Template *template.Template
	Data     interface{}
}

func (t *templateRender) Render(w http.ResponseWriter) error {
	t.WriteContentType(w)
	return t.Template.Execute(w, t.Data)
}

func (t *templateRender) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = []string{"text/html; charset=utf-8"}
	}
}

// LoadTemplates loads all templates with layout support.
func LoadTemplates() (*HTMLTemplates, error) {
	h := &HTMLTemplates{
		templates: make(map[string]*template.Template),
	}

	// Read the layout template
	layoutContent, err := templatesFS.ReadFile("templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("failed to read layout.html: %w", err)
	}

	// Walk through all templates
	err = fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".html" {
			return nil
		}

		// Skip layout.html itself
		if path == "templates/layout.html" {
			return nil
		}

		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		name := path[len("templates/"):]

		// Check if this is a partial (no layout needed)
		if filepath.Dir(path) == "templates/partials" {
			tmpl, err := template.New(name).Funcs(TemplateFuncs()).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse partial %s: %w", name, err)
			}
			h.templates[name] = tmpl
			return nil
		}

		// For regular templates, combine layout + content
		// First parse the layout
		tmpl, err := template.New("layout").Funcs(TemplateFuncs()).Parse(string(layoutContent))
		if err != nil {
			return fmt.Errorf("failed to parse layout for %s: %w", name, err)
		}

		// Then parse the page content (which defines "content" template)
		_, err = tmpl.Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", name, err)
		}

		h.templates[name] = tmpl

		return nil
	})

	if err != nil {
		return nil, err
	}

	return h, nil
}

// RenderTemplate renders a template to a bytes.Buffer (useful for testing).
func (h *HTMLTemplates) RenderTemplate(name string, data interface{}) (*bytes.Buffer, error) {
	h.mu.RLock()
	tmpl, ok := h.templates[name]
	h.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return nil, err
	}

	return buf, nil
}
