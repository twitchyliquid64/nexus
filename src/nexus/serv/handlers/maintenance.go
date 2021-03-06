package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/data"
	"nexus/data/integration"
	"nexus/data/session"
	"nexus/metrics"
	"nexus/serv/util"
	"os"
	"path"
	"time"
)

const (
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
	mux.HandleFunc("/admin/stats", h.StatsHandler)
	mux.HandleFunc("/admin/dobackup", h.BackupNowHandler)
	return nil
}

type statsData struct {
	BackupInfo map[string]interface{}
	Metrics    interface{}

	TableStats    map[string]data.TableStat
	TableCountErr error
}

// StatsHandler handles a HTTP request to /admin/stats
func (h *MaintenanceHandler) StatsHandler(response http.ResponseWriter, request *http.Request) {
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

	var templateData statsData
	templateData.TableStats, templateData.TableCountErr = data.GetTableStats(request.Context(), h.DB)
	templateData.Metrics = metrics.GetByCategory()
	templateData.BackupInfo = data.GetBackupStatistics()

	util.ApplyStrictTransportSecurity(request, response)
	util.LogIfErr("StatsHandler(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/statsResult.html"), templateData, response))
}

type cleanupData struct {
	NumLogsDeleted  int64
	LogsDeleteErr   error
	LogsCleanupTime time.Duration

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
	templateData.NumLogsDeleted, templateData.LogsDeleteErr = integration.DoLogsCleanup(request.Context(), h.DB)
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

	util.ApplyStrictTransportSecurity(request, response)
	util.LogIfErr("CleanupHandler(): %v", util.RenderPage(path.Join(h.TemplatePath, "templates/maintenanceResult.html"), templateData, response))
}

// BackupNowHandler handles a HTTP request to /admin/dobackup.
func (h *MaintenanceHandler) BackupNowHandler(response http.ResponseWriter, request *http.Request) {
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

	go func() {
		data.BackupNow()
	}()
	time.Sleep(time.Second)
	for data.GetBackupStatistics()["Dump in progress"].(bool) {
		time.Sleep(time.Second)
	}
	for data.GetBackupStatistics()["Upload in progress"].(bool) {
		time.Sleep(time.Second)
	}

	util.ApplyStrictTransportSecurity(request, response)
	http.Redirect(response, request, "/admin/stats", http.StatusTemporaryRedirect)
}
