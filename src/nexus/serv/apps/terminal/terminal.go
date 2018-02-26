package terminal

import (
	"context"
	"database/sql"
	//"encoding/json"
	"log"
	"net/http"
	//"nexus/data/fs"
	"nexus/data/session"
	"nexus/data/user"
	//intfs "nexus/fs"
	"nexus/serv/util"
	"os"
	"path"
	"strings"
	//"time"
)

// TerminalApp represents the terminal application.
type TerminalApp struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (a *TerminalApp) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	templatePath := ctx.Value("templatePath")
	if templatePath != nil {
		a.TemplatePath = templatePath.(string)
	} else {
		var err error
		a.TemplatePath, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	a.DB = db

	mux.HandleFunc("/app/terminal", a.Render)
	mux.HandleFunc("/app/terminal/query", a.HandleQuery)
	return nil
}

// Render generates page content.
func (a *TerminalApp) Render(response http.ResponseWriter, request *http.Request) {
	_, _, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	util.LogIfErr("TerminalApp.Render(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/terminal/main.html"), nil, response))
}

// HandleQuery is invoked to run a query.
func (a *TerminalApp) HandleQuery(response http.ResponseWriter, request *http.Request) {
	u, _, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	// var input struct {
	// 	Query string `json:"query"`
	// }
	// err = json.NewDecoder(request.Body).Decode(&input)
	// if err != nil {
	// 	log.Printf("json.Decode() Error: %v", err)
	// 	http.Error(response, "Internal server error", 500)
	// 	return
	// }

	log.Printf("RunQuery(%q): %v", request.FormValue("query"), RunSQL(request.Context(), a.DB, request.FormValue("query"), u.UID)) //TODO: input.Query
}

// EntryURL implements app.
func (a *TerminalApp) EntryURL() string {
	return "/app/terminal"
}

// Name implements app.
func (a *TerminalApp) Name() string {
	return "Terminal"
}

// Icon implements app.
func (a *TerminalApp) Icon() string {
	return "text_fields"
}

// ShouldShowIcon implements app.
func (a *TerminalApp) ShouldShowIcon(ctx context.Context, uid int) (bool, error) {
	attrs, err := user.GetAttrForUser(ctx, uid, a.DB)
	if err != nil {
		return false, err
	}
	for _, attr := range attrs {
		if strings.ToLower(attr.Name) == "terminal" {
			if strings.Contains(strings.ToLower(attr.Val), "no") || strings.Contains(strings.ToLower(attr.Val), "den") { //no or deny or denied or no access
				return false, nil
			}
			return true, nil
		}
	}
	return false, nil
}
