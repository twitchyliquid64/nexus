package mc

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data/mc"
	"nexus/data/session"
	"nexus/data/user"
	"nexus/serv/util"
	"os"
	"path"
	"strings"
	"time"
)

// ReconApp represents the recon application, as well as API endpoints for collecting data from entities in the field.
type ReconApp struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (a *ReconApp) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
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

	mux.HandleFunc("/app/recon", a.renderMainPage)
	mux.HandleFunc("/app/recon/status", a.handleStatus)
	mux.HandleFunc("/app/recon/heartbeat", a.handleHeartbeat)
	mux.HandleFunc("/app/recon/loc", a.handleLocationUpdate)
	mux.HandleFunc("/app/recon/api/status", a.serveStatusList)
	mux.HandleFunc("/app/recon/status/", a.renderStatusView)
	return nil
}

// returns true if user is authorized.
func (a *ReconApp) handleCheckAuthorized(response http.ResponseWriter, request *http.Request) bool {
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
		log.Printf("ReconApp.ShouldShowIcon() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return false
	}
	if !authorized {
		http.Error(response, "Unauthorized", 403)
		return false
	}
	return true
}

func (a *ReconApp) serveStatusList(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	type statusListRequest struct {
		UID, Limit, Offset int
	}
	var input statusListRequest
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&input)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	statuses, err := mc.ListStatus(request.Context(), input.UID, input.Limit, input.Offset, a.DB)
	b, err := json.Marshal(statuses)
	if err != nil {
		http.Error(response, err.Error(), 500)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

func (a *ReconApp) renderStatusView(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}
	util.LogIfErr("ReconApp.renderStatusView(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/mc_recon/statusView.html"), nil, response))
}

func (a *ReconApp) renderMainPage(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	type deviceInfo struct {
		Device        *mc.APIKey
		LocationCount int
		StatusCount   int

		LastLoc    time.Time
		Now        time.Time
		LastStatus time.Time
		Status     string
	}
	var templateData []deviceInfo

	devices, err := mc.GetAllEntityKeys(request.Context(), a.DB)
	if err != nil {
		log.Printf("mc.GetAllEntityKeys() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	for _, device := range devices {
		locCount, lastLoc, err := mc.LocationsCountForEntityRecent(request.Context(), device.UID, a.DB)
		if err != nil {
			log.Printf("mc.LocationsCountForEntityRecent() Error: %v", err)
			http.Error(response, "Internal server error", 500)
			return
		}
		statusCount, lastStatus, status, err := mc.RecentStatusInfoForEntity(request.Context(), device.UID, a.DB)
		if err != nil {
			log.Printf("mc.StatusesCountForEntityRecent() Error: %v", err)
			http.Error(response, "Internal server error", 500)
			return
		}
		templateData = append(templateData, deviceInfo{
			Device:        &device,
			LocationCount: locCount,
			LastLoc:       lastLoc,
			Now:           time.Now(),
			StatusCount:   statusCount,
			LastStatus:    lastStatus,
			Status:        status,
		})
	}

	util.LogIfErr("ReconApp.renderMainPage(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/mc_recon/main.html"), templateData, response))
}

// EntryURL implements app.
func (a *ReconApp) EntryURL() string {
	return "/app/recon"
}

// Name implements app.
func (a *ReconApp) Name() string {
	return "MC :: Recon"
}

// Icon implements app.
func (a *ReconApp) Icon() string {
	return "location_searching"
}

// ShouldShowIcon implements app.
func (a *ReconApp) ShouldShowIcon(ctx context.Context, uid int) (bool, error) {
	attrs, err := user.GetAttrForUser(ctx, uid, a.DB)
	if err != nil {
		return false, err
	}
	for _, attr := range attrs {
		if strings.ToLower(attr.Name) == "recon" {
			v := strings.ToLower(attr.Val)
			if strings.Contains(v, "yes") || strings.Contains(v, "allow") || strings.Contains(v, "true") {
				return true, nil
			}
		}
	}
	return false, nil
}
