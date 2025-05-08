package main

import (
	"net/http"
	"path/filepath"
	"text/template"
)

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	templates := template.Must(template.ParseFiles(
		filepath.Join("templates", "base.html"),
		filepath.Join("templates", tmpl+".html"),
	))
	templates.ExecuteTemplate(w, "base.html", data)
}
