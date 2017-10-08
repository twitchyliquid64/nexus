package apps

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
	"strconv"
	"strings"

	"github.com/twitchyliquid64/rwords"
)

// CodenameApp represents the codenames application.
type CodenameApp struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (a *CodenameApp) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
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

	mux.HandleFunc("/app/codename", a.Render)
	return nil
}

// Render generates page content.
func (a *CodenameApp) Render(response http.ResponseWriter, request *http.Request) {
	_, u, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	authorized, err := a.ShouldShowIcon(request.Context(), u.UID)
	if err != nil {
		log.Printf("CodenameApp.ShouldShowIcon() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	if !authorized {
		http.Error(response, "Unauthorized", 403)
		return
	}

	type suggestionRow struct {
		Title       string
		Suggestions []string
	}
	var out []suggestionRow

	for i := 0; i < 4; i++ {
		suggestions := make([]string, 8)
		for z := 0; z < 8; z++ {
			suggestions[z] = rwords.RandomSimple(i + 4)
		}

		out = append(out, suggestionRow{
			Title:       "Words with " + strconv.Itoa(i+4) + " sounds",
			Suggestions: suggestions,
		})
	}

	util.LogIfErr("CodenameApp.Render(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/codename/main.html"), out, response))
}

// EntryURL implements app.
func (a *CodenameApp) EntryURL() string {
	return "/app/codename"
}

// Name implements app.
func (a *CodenameApp) Name() string {
	return "Codename generator"
}

// Icon implements app.
func (a *CodenameApp) Icon() string {
	return "font_download"
}

// ShouldShowIcon implements app.
func (a *CodenameApp) ShouldShowIcon(ctx context.Context, uid int) (bool, error) {
	attrs, err := user.GetAttrForUser(ctx, uid, a.DB)
	if err != nil {
		return false, err
	}
	for _, attr := range attrs {
		if strings.ToLower(attr.Name) == "codenamegenerator" {
			if strings.Contains(strings.ToLower(attr.Val), "no") || strings.Contains(strings.ToLower(attr.Val), "den") { //no or deny or denied or no access
				return false, nil
			}
		}
	}
	return true, nil
}
