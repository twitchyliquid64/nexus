package util

import (
	"html/template"
	"net/http"
	"path"
)

// RenderPage renders the template at the specific path.
func RenderPage(pagePath string, pageData interface{}, rw http.ResponseWriter) error {
	t, err := template.New("").Delims("{!{", "}!}").ParseFiles(pagePath)
	if err != nil {
		http.Error(rw, "Internal Server Error", 500)
		return err
	}
	return t.ExecuteTemplate(rw, path.Base(pagePath), pageData)
}
