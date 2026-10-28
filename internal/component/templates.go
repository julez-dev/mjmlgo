package component

import (
	"embed"
	"sync"
	"text/template"
)

//go:embed texttemplate
var templateFS embed.FS

var (
	templates *template.Template
	o         = &sync.Once{}
)

func init() {
	o.Do(func() {
		t, err := template.ParseFS(templateFS, "texttemplate/*.tmpl")
		if err != nil {
			panic(err)
		}

		templates = t
	})
}
