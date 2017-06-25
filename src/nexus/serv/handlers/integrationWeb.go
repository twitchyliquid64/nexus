package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"nexus/data/integration"
	"nexus/serv/util"
)

// IntegrationHandler handles HTTP endpoints for the integrations UI.
type IntegrationHandler struct {
	DB *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *IntegrationHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db

	mux.HandleFunc("/web/v1/integrations/mine", h.HandleGetMine)
	mux.HandleFunc("/web/v1/integrations/create/runnable", h.HandleCreateRunnable)
	mux.HandleFunc("/web/v1/integrations/delete/runnable", h.HandleDeleteRunnable)
	return nil
}

// HandleCreateRunnable handles web requests to create a runnable.
func (h *IntegrationHandler) HandleCreateRunnable(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var runnable integration.Runnable
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&runnable)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}
	runnable.OwnerID = usr.UID

	err = integration.DoCreateRunnable(request.Context(), &runnable, h.DB)
	if util.InternalHandlerError("integration.DoCreateRunnable(struct)", response, request, err) {
		return
	}
}

// HandleDeleteRunnable handles web requests to delete a runnable.
func (h *IntegrationHandler) HandleDeleteRunnable(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var IDs []int
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&IDs)
	if util.InternalHandlerError("json.Decode([]int)", response, request, err) {
		return
	}

	for _, id := range IDs {
		runnable, errGetRunnable := integration.GetRunnable(request.Context(), id, h.DB)
		if util.InternalHandlerError("integration.GetRunnable(int)", response, request, errGetRunnable) {
			return
		}
		if runnable.OwnerID != usr.UID {
			http.Error(response, "You do not own this integration.", 403)
			return
		}
	}

	for _, id := range IDs {
		err = integration.DoDeleteRunnable(request.Context(), id, h.DB)
		if util.InternalHandlerError("integration.DoDeleteRunnable(int)", response, request, err) {
			return
		}
	}
}

// HandleGetMine handles web requests to retrieve the integrations owned by an account.
func (h *IntegrationHandler) HandleGetMine(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	integrations, err := integration.GetAllForUser(request.Context(), usr.UID, h.DB)
	if util.InternalHandlerError("integration.GetAllForUser()", response, request, err) {
		return
	}

	for _, i := range integrations {
		i.Triggers, err = integration.GetTriggersForRunnable(request.Context(), i.UID, h.DB)
		if util.InternalHandlerError("integration.GetTriggersForRunnable()", response, request, err) {
			return
		}
	}

	b, err := json.Marshal(integrations)
	if util.InternalHandlerError("json.Marshal(integrations)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
