package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data/session"
	"nexus/data/user"
	nexus_apps "nexus/serv/apps"
	"nexus/serv/apps/mc"
	"nexus/serv/apps/terminal"
	"nexus/serv/util"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
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
	&nexus_apps.MediaApp{},
	&mc.ReconApp{},
	&terminal.TerminalApp{},
}

// appsInternalHandler is a special-case handler to serve the list of apps a user can access.
// It is in the package (rather than handlers/) so it can access the apps global.
type appsInternalHandler struct {
	db *sql.DB
}

func (h *appsInternalHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.db = db
	mux.HandleFunc("/apps/list", h.serveAppsListForUser)
	mux.HandleFunc("/apps/getJWT", h.createJWTForUser)
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
		Name  string
		Kind  int
		Icon  string
		URL   string
		Extra string
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
				Kind: user.ExternAppURLKind,
			})
		}
	}
	extApps, err := user.GetExtAppsForUser(request.Context(), u.UID, h.db)
	if err != nil {
		http.Error(response, "Internal server error", 500)
		log.Printf("GetExtAppsForUser(%q) Error: %v", u.Username, err)
		return
	}
	for _, app := range extApps {
		out = append(out, appInfo{
			Name: app.Name,
			Icon: app.Icon,
			URL:  app.Val,
			Kind: app.Kind,
		})
	}

	b, err := json.Marshal(out)
	if err != nil {
		http.Error(response, err.Error(), 500)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

type jwtClaim struct {
	UID      int    `json:"uid"`
	Username string `json:"username"`
	AppName  string `json:"app_name"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.StandardClaims
}

func (h *appsInternalHandler) createJWTForUser(response http.ResponseWriter, request *http.Request) {
	_, u, authErr := util.AuthInfo(request, h.db)
	if authErr == session.ErrInvalidSession || authErr == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if authErr != nil {
		log.Printf("AuthInfo() Error: %s", authErr)
		http.Error(response, "Internal server error", 500)
		return
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		log.Printf("json.Decode() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	extApps, err := user.GetExtAppsForUser(request.Context(), u.UID, h.db)
	if err != nil {
		http.Error(response, "Internal server error", 500)
		log.Printf("GetExtAppsForUser(%q) Error: %v", u.Username, err)
		return
	}

	for _, app := range extApps {
		if app.Name == input.Name && app.Kind == user.ExternAppJWTKind {
			// get the secret.
			var extra map[string]string
			if err := json.Unmarshal([]byte(app.Extra), &extra); err != nil {
				log.Printf("json.Unmarshal() Error: %v", err)
				http.Error(response, "Internal server error", 500)
				return
			}

			// build and sign the assertion.
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaim{
				UID:      u.UID,
				Username: u.Username,
				IsAdmin:  u.AdminPerms.Accounts,
				AppName:  app.Name,
				StandardClaims: jwt.StandardClaims{
					Issuer:    "NEXUS",
					ExpiresAt: time.Now().Add(120 * time.Second).Unix(),
				},
			})
			ss, err := token.SignedString([]byte(extra["secret"]))
			if err != nil {
				http.Error(response, "Internal server error", 500)
				log.Printf("token.SignedString() Error: %v", err)
				return
			}
			response.Write([]byte(ss))
			return
		}
	}
	http.Error(response, "No such JWT app", http.StatusBadRequest)
}
