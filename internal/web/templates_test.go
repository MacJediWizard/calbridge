package web

import (
	"strings"
	"testing"
)

func TestLoadTemplates(t *testing.T) {
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates() error = %v", err)
	}

	if templates == nil {
		t.Fatal("LoadTemplates() returned nil")
	}

	// Check that expected templates are loaded
	expectedTemplates := []string{
		"login.html",
		"dashboard.html",
		"error.html",
		"logs.html",
		"sources/list.html",
		"sources/add.html",
		"sources/edit.html",
		"partials/error.html",
		"partials/sync_triggered.html",
	}

	for _, name := range expectedTemplates {
		if _, ok := templates.templates[name]; !ok {
			t.Errorf("Template %q not found", name)
		}
	}
}

func TestRenderLoginTemplate(t *testing.T) {
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates() error = %v", err)
	}

	data := map[string]interface{}{
		"Title": "Test Login",
	}

	buf, err := templates.RenderTemplate("login.html", data)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	html := buf.String()

	// Check that the layout is included
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Expected HTML doctype in output")
	}

	// Check that the content is included
	if !strings.Contains(html, "CalBridge") {
		t.Error("Expected 'CalBridge' in output")
	}

	if !strings.Contains(html, "Sign in with SSO") {
		t.Error("Expected 'Sign in with SSO' button in output")
	}

	// Check that Tailwind CSS is loaded
	if !strings.Contains(html, "tailwindcss.com") {
		t.Error("Expected Tailwind CSS script in output")
	}
}

func TestRenderPartialTemplate(t *testing.T) {
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("LoadTemplates() error = %v", err)
	}

	data := map[string]interface{}{
		"error": "Test error message",
	}

	buf, err := templates.RenderTemplate("partials/error.html", data)
	if err != nil {
		t.Fatalf("RenderTemplate() error = %v", err)
	}

	html := buf.String()

	// Partials should NOT include the layout
	if strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("Partial should not include DOCTYPE")
	}

	// But should include the error message
	if !strings.Contains(html, "Test error message") {
		t.Error("Expected error message in output")
	}
}
