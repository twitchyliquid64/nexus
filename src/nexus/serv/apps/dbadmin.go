package apps

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data"
	"nexus/data/session"
	"nexus/data/user"
	"nexus/serv/util"
	"os"
	"path"
)

// DatabaseAdminApp represents the database administration application.
type DatabaseAdminApp struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (a *DatabaseAdminApp) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
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

	mux.HandleFunc("/app/dbadmin", a.Render)
	return nil
}

// returns true if user is authorized.
func (a *DatabaseAdminApp) handleCheckAuthorized(response http.ResponseWriter, request *http.Request) bool {
	_, u, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return false
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return false
	}

	authorized, err := a.ShouldShowIcon(request.Context(), u.UID)
	if err != nil {
		log.Printf("DatabaseAdminApp.ShouldShowIcon() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return false
	}
	if !authorized {
		http.Error(response, "Unauthorized", 403)
		return false
	}
	return true
}

// Render generates page content.
func (a *DatabaseAdminApp) doQuery(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	type queryResult struct {
		Error   string
		Columns []string
	}
	var result queryResult

	type queryRequest struct {
		Query string
	}
	var input queryRequest
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&input)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	c := data.GetSQLiteConn()
	rows, err := c.Query(input.Query, nil)
	if err != nil {
		result.Error = err.Error()
	} else {
		defer rows.Close()
		result.Columns = rows.Columns()
		for rows.
	}
}

// Render generates page content.
func (a *DatabaseAdminApp) Render(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	util.LogIfErr("DatabaseAdminApp.Render(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/dbadmin/main.html"), nil, response))
}

// EntryURL implements app.
func (a *DatabaseAdminApp) EntryURL() string {
	return "/app/dbadmin"
}

// Name implements app.
func (a *DatabaseAdminApp) Name() string {
	return "Database Admin"
}

// Icon implements app.
func (a *DatabaseAdminApp) Icon() string {
	return "storage"
}

// ShouldShowIcon implements app.
func (a *DatabaseAdminApp) ShouldShowIcon(ctx context.Context, uid int) (bool, error) {
	usr, err := user.GetByUID(ctx, uid, a.DB)
	if err != nil {
		return false, err
	}
	if usr.AdminPerms.Data {
		return true, nil
	}
	return false, nil
}
