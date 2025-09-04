package templates

import (
	_ "embed"
	"html/template"
	"io"
)

//go:embed dashboard.html
var dashboardHTML string

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