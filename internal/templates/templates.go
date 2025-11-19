package templates

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net/http"
)

var (
	//go:embed dashboard.html
	dashboardHTML string

	//go:embed static/*
	staticFiles embed.FS
)

var staticHTTPFS http.FileSystem

func init() {
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic("failed to load embedded static files: " + err.Error())
	}
	staticHTTPFS = http.FS(sub)
}

// DashboardData contains the data needed for the dashboard template
type DashboardData struct {
	DashboardSecret string
}

// Dashboard represents the dashboard template
type Dashboard struct {
	template *template.Template
}

// NewDashboard creates a new dashboard template instance
func NewDashboard() (*Dashboard, error) {
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		return nil, err
	}

	return &Dashboard{
		template: tmpl,
	}, nil
}

// Render executes the dashboard template with the given data
func (d *Dashboard) Render(w io.Writer, data DashboardData) error {
	return d.template.Execute(w, data)
}

// StaticFileSystem returns an http.FileSystem for serving embedded static assets.
func StaticFileSystem() http.FileSystem {
	return staticHTTPFS
}
