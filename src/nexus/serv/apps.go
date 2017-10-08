package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data/session"
	nexus_apps "nexus/serv/apps"
	"nexus/serv/util"
)

type app interface {
	Name() string
	Icon() string
	EntryURL() string
	ShouldShowIcon(ctx context.Context, uid int) (bool, error)
	BindMux(context.Context, *http.ServeMux, *sql.DB) error
}

var apps = []app{
	&nexus_apps.CodenameApp{},
	&nexus_apps.YtdlApp{},
}

// appsInternalHandler is a special-case handler to serve the list of apps a user can access.
// It is in the package (rather than handlers/) so it can access the apps global.
type appsInternalHandler struct {
	db *sql.DB
}

func (h *appsInternalHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.db = db
	mux.HandleFunc("/apps/list", h.serveAppsListForUser)
	return nil
}

func (h *appsInternalHandler) serveAppsListForUser(response http.ResponseWriter, request *http.Request) {
	_, u, authErr := util.AuthInfo(request, h.db)
	if authErr == session.ErrInvalidSession || authErr == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if authErr != nil {
		log.Printf("AuthInfo() Error: %s", authErr)
		http.Error(response, "Internal server error", 500)
		return
	}

	type appInfo struct {
		Name string
		Icon string
		URL  string
	}
	var out []appInfo

	for _, app := range apps {
		shouldShow, err := app.ShouldShowIcon(request.Context(), u.UID)
		if err != nil {
			http.Error(response, "Internal server error", 500)
			log.Printf("(%s).ShouldShowIcon(%s) Error: %v", app.Name(), u.Username, err)
			return
		}
		if shouldShow {
			out = append(out, appInfo{
				Name: app.Name(),
				Icon: app.Icon(),
				URL:  app.EntryURL(),
			})
		}
	}

	b, err := json.Marshal(out)
	if err != nil {
		http.Error(response, err.Error(), 500)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
