package forms

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"nexus/data"
	"path"
)

type source interface {
	Title() string
	Description() string
	UniqueID() string
	IsAdminOnly() bool
	Actions() []interface{}
}

type form interface {
	Title() string
	Icon() string
	UniqueID() string
	FormFields() []interface{}
	OnSubmitHandler() func(context.Context, map[string]string, int, *sql.DB) error
}

type field interface {
	Type() string
	Label() string
	UniqueID() string
	ValidationRegex() string
}

func validate(s source) (int, error) {
	for i, f := range s.Actions() {
		if okForm, ok := f.(form); ok {
			for x, ff := range okForm.FormFields() {
				if _, fieldOk := ff.(field); !fieldOk {
					return i, fmt.Errorf("field at index %d does not implement field interface", x)
				}
			}
		} else {
			return i, errors.New("does not implement form interface")
		}
	}
	return 0, nil
}

func getForms(adminOnly bool) []source {
	forms := data.GetForms()
	var out []source
	for _, form := range forms {
		errorIndex, validationErr := validate(form)
		if validationErr != nil {
			log.Printf("[forms] Form %d from source %q failed validation: %s", errorIndex, form.UniqueID(), validationErr.Error())
			continue
		}
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

func renderForms(forms []source) (*bytes.Buffer, error) {
	var buff bytes.Buffer
	return &buff, render("templates/forms/form.html", &buff, forms)
}

// Render is called to produce a form for a given user.
func Render(ctx context.Context, adminOnly bool, userID int, w io.Writer) error {
	forms := getForms(adminOnly)
	listBuffer, err := renderList(forms)
	if err != nil {
		return err
	}
	formsBuffer, err := renderForms(forms)
	if err != nil {
		return err
	}

	return render("templates/forms/base.html", w, struct {
		List  template.HTML
		Forms template.HTML
	}{
		List:  template.HTML(listBuffer.String()),
		Forms: template.HTML(formsBuffer.String()),
	})
}
