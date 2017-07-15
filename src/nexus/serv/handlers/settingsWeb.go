package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/data/session"
	"nexus/forms"
	"nexus/serv/util"
)

// SettingsHandler handles endpoints which are used to change settings
type SettingsHandler struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *SettingsHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db

	mux.HandleFunc("/settings/show", h.Render)
	mux.HandleFunc("/settings/action/do/", h.HandleSubmission)
	return nil
}

// HandleSubmission handles HTTP requests to submit forms.
func (h *SettingsHandler) HandleSubmission(response http.ResponseWriter, request *http.Request) {
	_, u, err := util.AuthInfo(request, h.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	err = forms.HandleSubmission(request, request.URL.Path[len("/settings/action/do/"):], u.UID, h.DB)
	if err != nil {
		log.Printf("forms.HandleSubmission() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	http.Redirect(response, request, "/settings/show", 302)
}

// Render handles a HTTP request to render a settings list
func (h *SettingsHandler) Render(response http.ResponseWriter, request *http.Request) {
	_, u, err := util.AuthInfo(request, h.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	if !u.AdminPerms.Accounts {
		http.Error(response, "Not authorized", 403)
		return
	}

	err = forms.Render(request.Context(), false, u.UID, response, h.DB)
	if err != nil {
		log.Printf("forms.Render() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}
}