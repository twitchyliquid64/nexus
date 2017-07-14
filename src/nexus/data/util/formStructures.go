package util

// FormDescriptor describes a form which can be rendered to a user.
type FormDescriptor struct {
	FormTitle string
	ID        string
	AdminOnly bool
}

func (f *FormDescriptor) Title() string {
	return f.FormTitle
}
func (f *FormDescriptor) UniqueID() string {
	return f.ID
}
func (f *FormDescriptor) IsAdminOnly() bool {
	return f.AdminOnly
}
