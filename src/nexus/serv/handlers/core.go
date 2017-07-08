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
	mux.HandleFunc("/logout", h.HandleLogout)
	return nil
}

// HandleIndex handles a HTTP request to /.
func (h *CoreHandler) HandleIndex(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.Error(response, "Not Found", 404)
		return
	}

	_, u, err := util.AuthInfo(request, h.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	util.LogIfErr("HandleIndex(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/index.html"), u, response))
}

// HandleLogout handles a HTTP request to /logout.
func (h *CoreHandler) HandleLogout(response http.ResponseWriter, request *http.Request) {
	s, _, err := util.AuthInfo(request, h.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	revokeErr := session.Revoke(request.Context(), s.SID, h.DB)
	if revokeErr != nil {
		http.Error(response, "Failed to revoke session", 500)
		return
	}
	http.Redirect(response, request, "/", 303)
}

type loginTemplateData struct {
	Msg      string
	ShowOTP  bool
	Username string
	Password string
}

// HandleLogin handles a HTTP request to /login.
func (h *CoreHandler) HandleLogin(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	if request.Method == "GET" {
		util.LogIfErr("HandleLogin(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/login.html"), loginTemplateData{Msg: request.FormValue("msg")}, response))
	}

	if request.Method == "POST" {
		if err := request.ParseForm(); err != nil {
			http.Error(response, "Could not parse form", 400)
			return
		}
		ok, authDetails, err := util.CheckAuth(ctx, request, h.DB)
		log.Printf("Attempted auth for %s: %+v", request.FormValue("user"), authDetails)
		if err != user.ErrUserDoesntExist && util.InternalHandlerError("checkAuth()", response, request, err) {
			return
		}
		if ok {
			log.Printf("Got correct credentials for %s using %+v, creating session", request.FormValue("user"), authDetails)
			usr, err := user.Get(ctx, request.FormValue("user"), h.DB)
			if util.InternalHandlerError("user.Get()", response, request, err) {
				return
			}

			if usr.IsRobot {
				log.Printf("Robot account %q attempted to login to web interface, login denied", usr.Username)
				http.Error(response, "Access Denied", 403)
				return
			}

			sid, err := session.Create(ctx, usr.UID, true, true, session.AuthPass, h.DB)
			if util.InternalHandlerError("session.Create()", response, request, err) {
				return
			}
			http.SetCookie(response, &http.Cookie{Name: "sid", Value: sid})
			http.Redirect(response, request, "/", 303)
		} else {
			if authDetails.OTPWanted {
				tData := loginTemplateData{
					ShowOTP:  true,
					Username: request.FormValue("user"),
					Password: request.FormValue("password"),
				}
				util.LogIfErr("HandleLogin(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/login.html"), tData, response))
				return
			}

			http.Redirect(response, request, "/login?msg=Invalid%20credentials,%20please%20try%20again.", 303) //303 = must GET
		}
	}
}
