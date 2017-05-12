package util

import (
	"html/template"
	"net/http"
)

// RenderPage renders the template at the specific path.
func RenderPage(pagePath string, pageData interface{}, rw http.ResponseWriter) error {
	t, err := template.ParseFiles(pagePath)
	if err != nil {
		return err
	}
	return t.Execute(rw, pageData)
}
