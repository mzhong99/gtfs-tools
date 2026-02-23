package gtfs_web

import (
	"embed"
	"io"
	"text/template"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Renderer struct {
	tmpl *template.Template
}

func NewRenderer() (*Renderer, error) {
	tmpl, err := template.New("root").ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &Renderer{tmpl: tmpl}, nil
}

func (renderer *Renderer) Render(writer io.Writer, name string, data any) error {
	return renderer.tmpl.ExecuteTemplate(writer, name, data)
}
