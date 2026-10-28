package component

import (
	"embed"
	"text/template"
)

//go:embed texttemplate
var templateFS embed.FS

var templates *template.Template

func init() {
	parsed, err := template.ParseFS(templateFS, "texttemplate/*.tmpl")
	if err != nil {
		panic(err)
	}

	templates = parsed
}
