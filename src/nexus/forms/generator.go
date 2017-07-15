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
	GetContentSections() []interface{}
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

type table interface {
	Title() string
	Description() string
	UniqueID() string
	ColNames() []string
	OnLoadHandler() func(context.Context, int, *sql.DB) ([]interface{}, error)
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
	for i, f := range s.GetContentSections() {
		if _, ok := f.(table); !ok {
			return i, errors.New("does not implement table interface")
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
			log.Printf("[forms] Form/table %d from source %q failed validation: %s", errorIndex, form.UniqueID(), validationErr.Error())
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

func renderForms(forms []source, contents map[string][]interface{}, errs map[string]error) (*bytes.Buffer, error) {
	var buff bytes.Buffer
	return &buff, render("templates/forms/form.html", &buff, struct {
		Sources     []source
		TableData   map[string][]interface{}
		TableErrors map[string]error
	}{
		Sources:     forms,
		TableData:   contents,
		TableErrors: errs,
	})
}

func computeTableContents(ctx context.Context, forms []source, userID int, db *sql.DB) (map[string][]interface{}, map[string]error) {
	contents := map[string][]interface{}{}
	errs := map[string]error{}

	for _, s := range forms {
		for _, section := range s.GetContentSections() {
			t := section.(table)
			result, err := t.OnLoadHandler()(ctx, userID, db)
			if err != nil {
				errs[t.UniqueID()] = err
			} else {
				contents[t.UniqueID()] = result
			}
		}
	}
	return contents, errs
}

// Render is called to produce a form for a given user.
func Render(ctx context.Context, adminOnly bool, userID int, w io.Writer, db *sql.DB) error {
	forms := getForms(adminOnly)
	listBuffer, err := renderList(forms)
	if err != nil {
		return err
	}
	sectionData, sectionErrs := computeTableContents(ctx, forms, userID, db)
	formsBuffer, err := renderForms(forms, sectionData, sectionErrs)
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
