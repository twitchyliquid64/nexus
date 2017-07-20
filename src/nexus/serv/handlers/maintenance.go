package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/data"
	"nexus/data/integration"
	"nexus/data/session"
	"nexus/serv/util"
	"os"
	"path"
	"time"
)

const (
	maxLogsRetentionDays    = 6
	maxSessionLengthDays    = 14
	maxSessionRetentionDays = 28
)

// MaintenanceHandler handles endpoints which represent maintainence operations.
type MaintenanceHandler struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *MaintenanceHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
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

	mux.HandleFunc("/admin/cleanup", h.CleanupHandler)
	return nil
}

type cleanupData struct {
	LogsRetentionDays int
	NumLogsDeleted    int64
	LogsDeleteErr     error
	LogsCleanupTime   time.Duration

	MaxSessionDays     int
	NumSessionsRevoked int64
	SessionRevokeErr   error
	SessionRevokeTime  time.Duration

	MaxSessionRetentionDays int
	NumSessionsDeleted      int64
	SessionDeleteErr        error
	SessionDeleteTime       time.Duration

	VacuumErr  error
	VacuumTime time.Duration
}

// CleanupHandler handles a HTTP request to /admin/cleanup.
func (h *MaintenanceHandler) CleanupHandler(response http.ResponseWriter, request *http.Request) {
	var templateData cleanupData

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

	logsCleanupStart := time.Now()
	templateData.LogsRetentionDays = maxLogsRetentionDays
	templateData.NumLogsDeleted, templateData.LogsDeleteErr = integration.DoLogsCleanup(request.Context(), maxLogsRetentionDays, h.DB)
	templateData.LogsCleanupTime = time.Now().Sub(logsCleanupStart)

	sessionRevokeStart := time.Now()
	templateData.MaxSessionDays = maxSessionLengthDays
	templateData.NumSessionsRevoked, templateData.SessionRevokeErr = session.RevokeByAge(request.Context(), maxSessionLengthDays, h.DB)
	templateData.SessionRevokeTime = time.Now().Sub(sessionRevokeStart)

	sessionDeleteStart := time.Now()
	templateData.MaxSessionRetentionDays = maxSessionRetentionDays
	templateData.NumSessionsDeleted, templateData.SessionDeleteErr = session.DeleteRevokedByAge(request.Context(), maxSessionRetentionDays, h.DB)
	templateData.SessionDeleteTime = time.Now().Sub(sessionDeleteStart)

	vacuumStart := time.Now()
	templateData.VacuumErr = data.Vacuum(h.DB)
	templateData.VacuumTime = time.Now().Sub(vacuumStart)

	util.LogIfErr("CleanupHandler(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/maintenanceResult.html"), templateData, response))
}
