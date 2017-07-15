package util

import (
	"context"
	"database/sql"
)

// FormDescriptor describes a settings page which can be rendered to a user.
type FormDescriptor struct {
	SettingsTitle string
	ID            string
	Desc          string
	AdminOnly     bool
	Forms         []*ActionDescriptor
	Tables        []*TableDescriptor
}

// Title returns the title of the form.
func (f *FormDescriptor) Title() string {
	return f.SettingsTitle
}

// UniqueID returns a URL-safe unique identifier for this form.
func (f *FormDescriptor) UniqueID() string {
	return f.ID
}

// IsAdminOnly returns true if the form should only be presented to system administrators.
func (f *FormDescriptor) IsAdminOnly() bool {
	return f.AdminOnly
}

// Description returns text to be displayed at the top of the settings page.
func (f *FormDescriptor) Description() string {
	return f.Desc
}

// Actions returns a list of forms.
func (f *FormDescriptor) Actions() []interface{} {
	out := make([]interface{}, len(f.Forms))
	for i, f := range f.Forms {
		out[i] = f
	}
	return out
}

// GetContentSections returns a list of tables.
func (f *FormDescriptor) GetContentSections() []interface{} {
	out := make([]interface{}, len(f.Tables))
	for i, f := range f.Tables {
		out[i] = f
	}
	return out
}

// TableDescriptor represents a table surfaced in the settings.
type TableDescriptor struct {
	Name         string
	ID           string
	Desc         string
	Cols         []string
	FetchContent func(context.Context, int, *sql.DB) ([]interface{}, error)
}

// Title returns the title of the table.
func (f *TableDescriptor) Title() string {
	return f.Name
}

// UniqueID returns a URL-safe unique identifier for this table.
func (f *TableDescriptor) UniqueID() string {
	return f.ID
}

// Description returns text to be displayed at the top of the table.
func (f *TableDescriptor) Description() string {
	return f.Desc
}

// ColNames returns the column names of the table.
func (f *TableDescriptor) ColNames() []string {
	return f.Cols
}

// OnLoadHandler returns the content to populate the table.
func (f *TableDescriptor) OnLoadHandler() func(context.Context, int, *sql.DB) ([]interface{}, error) {
	return f.FetchContent
}

// ActionDescriptor represents a form which can be submitted.
type ActionDescriptor struct {
	Name     string
	ID       string
	IcoStr   string
	Fields   []*Field
	OnSubmit func(context.Context, map[string]string, int, *sql.DB) error
}

// Title returns the title of the form.
func (f *ActionDescriptor) Title() string {
	return f.Name
}

// UniqueID returns a URL-safe unique identifier for this form.
func (f *ActionDescriptor) UniqueID() string {
	return f.ID
}

// Icon returns a material-icons icon string.
func (f *ActionDescriptor) Icon() string {
	if f.IcoStr == "" {
		return "add"
	}
	return f.IcoStr
}

// FormFields returns a list of form fields.
func (f *ActionDescriptor) FormFields() []interface{} {
	out := make([]interface{}, len(f.Fields))
	for i, f := range f.Fields {
		out[i] = f
	}
	return out
}

// OnSubmitHandler returns a function to be invoked when a form is submitted.
func (f *ActionDescriptor) OnSubmitHandler() func(context.Context, map[string]string, int, *sql.DB) error {
	return f.OnSubmit
}

// Field represents a input in a form.
type Field struct {
	Kind              string
	Name              string
	ID                string
	ValidationPattern string
	SelectOptions     map[string]string
}

// Type returns what type of field the struct represents.
func (f *Field) Type() string {
	return f.Kind
}

// Label returns what type of field the struct represents.
func (f *Field) Label() string {
	return f.Name
}

// UniqueID returns a URL-safe unique identifier for this field.
func (f *Field) UniqueID() string {
	return f.ID
}

// ValidationRegex returns a regex that must match for the form to be valid.
func (f *Field) ValidationRegex() string {
	return f.ValidationPattern
}

// Options returns the set of options for a select field.
func (f *Field) Options() map[string]string {
	return f.SelectOptions
}
