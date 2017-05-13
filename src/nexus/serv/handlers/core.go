package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/data/session"
	"nexus/data/user"
	"nexus/serv/util"
	"os"
	"path"
)

// CoreHandler handles feature-critical HTTP endpoints such as authentication
type CoreHandler struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *CoreHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	templatePath := ctx.Value("templatePath")
	if templatePath != nil {
		h.TemplatePath = templatePath.(string)
	} else {
		var err error
		h.TemplatePath, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	h.DB = db

	mux.HandleFunc("/", h.HandleIndex)
	mux.HandleFunc("/login", h.HandleLogin)
	return nil
}

// HandleIndex handles a HTTP request to /.
func (h *CoreHandler) HandleIndex(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.Error(response, "Not Found", 404)
		return
	}

	_, _, err := util.AuthInfo(request, h.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	util.LogIfErr("HandleIndex(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/index.html"), nil, response))
}

// HandleLogin handles a HTTP request to /login.
func (h *CoreHandler) HandleLogin(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	if request.Method == "GET" {
		util.LogIfErr("HandleLogin(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/login.html"), nil, response))
	}

	if request.Method == "POST" {
		if err := request.ParseForm(); err != nil {
			http.Error(response, "Could not parse form", 400)
			return
		}
		ok, err := user.CheckBasicAuth(ctx, request.FormValue("user"), request.FormValue("password"), h.DB)
		if err != nil {
			http.Error(response, "Internal server error", 500)
			log.Printf("CheckBasicAuth() Error: %s", err)
			return
		}
		if ok {
			log.Printf("Got correct basicpass credentials for %s, creating session", request.FormValue("user"))
			usr, err := user.Get(ctx, request.FormValue("user"), h.DB)
			if err != nil {
				http.Error(response, "Internal server error", 500)
				log.Printf("user.Get() Error: %s", err)
				return
			}

			sid, err := session.Create(ctx, usr.UID, true, false, session.AuthPass, h.DB)
			if err != nil {
				http.Error(response, "Internal server error", 500)
				log.Printf("session.Create() Error: %s", err)
				return
			}
			http.SetCookie(response, &http.Cookie{Name: "sid", Value: sid})
			http.Redirect(response, request, "/", 303)
		} else {
			http.Redirect(response, request, "/login", 303) //303 = must GET
		}
	}
}
