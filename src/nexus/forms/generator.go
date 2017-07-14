package forms

import (
	"bytes"
	"context"
	"html/template"
	"io"
	"nexus/data"
	"path"
)

type source interface {
	Title() string
	UniqueID() string
	IsAdminOnly() bool
}

func getForms(adminOnly bool) []source {
	forms := data.GetForms()
	var out []source
	for _, form := range forms {
		switch form.IsAdminOnly() {
		case true:
			if adminOnly {
				out = append(out, form)
			}
		default:
			out = append(out, form)
		}
	}
	return out
}

func render(p string, w io.Writer, data interface{}) error {
	t, err := template.New("").Delims("{!{", "}!}").ParseFiles(p)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, path.Base(p), data)
}

func renderList(forms []source) (*bytes.Buffer, error) {
	var buff bytes.Buffer
	return &buff, render("templates/forms/list.html", &buff, forms)
}

// Render is called to produce a form for a given user.
func Render(ctx context.Context, adminOnly bool, userID int, w io.Writer) error {
	forms := getForms(adminOnly)
	listBuffer, err := renderList(forms)
	if err != nil {
		return err
	}

	return render("templates/forms/base.html", w, struct {
		List template.HTML
	}{
		List: template.HTML(listBuffer.String()),
	})
}
