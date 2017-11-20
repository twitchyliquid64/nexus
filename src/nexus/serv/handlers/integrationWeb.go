package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data/integration"
	integrationState "nexus/integration"
	notify "nexus/integration/log"
	"nexus/serv/util"
	"time"

	"golang.org/x/net/websocket"

	"github.com/robertkrimen/otto"
)

const (
	// maxAgeRuns is the max age of a run before it no longer appears in the list.
	maxAgeRuns = time.Hour * 24 * 21
	// max querible time.
	maxAgeLogs = time.Hour * 24 * 180
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
	mux.HandleFunc("/web/v1/integrations/edit/runnable", h.HandleEditRunnable)
	mux.HandleFunc("/web/v1/integrations/code/save", h.HandleSaveCode)
	mux.HandleFunc("/web/v1/integrations/run/manual", h.HandleRun)
	mux.HandleFunc("/web/v1/integrations/log/runs", h.HandleGetRuns)
	mux.HandleFunc("/web/v1/integrations/log/entries", h.HandleGetLogs)
	mux.Handle("/web/v1/integrations/log/stream/", websocket.Handler(h.HandleLogsWS))
	return nil
}

type logsConsumer struct {
	c        *websocket.Conn
	doneChan chan bool
}

func (l *logsConsumer) Done() {
	close(l.doneChan)
}
func (l *logsConsumer) Message(msg *integration.Log) {
	b, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[logsConsumer] json.Marshal() error: %v", err)
		return
	}
	l.c.Write(b)
}

// HandleLogsWS provides realtime logs for a given runnable.
func (h *IntegrationHandler) HandleLogsWS(c *websocket.Conn) {
	s, usr, err := util.AuthInfo(c.Request(), h.DB)
	if err != nil {
		log.Printf("HandleLogWS Auth error: %v", err)
		return
	}
	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		return
	}

	consumer := logsConsumer{
		c:        c,
		doneChan: make(chan bool),
	}
	log.Printf("Streaming logs for: %s", c.Request().URL.Path[len("/web/v1/integrations/log/stream/"):])
	err = notify.Subscribe(c.Request().URL.Path[len("/web/v1/integrations/log/stream/"):], &consumer)
	if err != nil {
		log.Printf("HandleLogWS notify.Subscribe() error: %v", err)
		return
	}
	<-consumer.doneChan
}

// HandleCreateRunnable handles web requests to create a runnable.
func (h *IntegrationHandler) HandleCreateRunnable(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
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

	err = integrationState.RunnableChanged(&runnable)
	if util.InternalHandlerError("integrationState.RunnableChanged(runnable)", response, request, err) {
		return
	}
}

// HandleSaveCode handles web requests to save the code of a runnable.
func (h *IntegrationHandler) HandleSaveCode(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
		return
	}

	var details struct {
		UID  int
		Code string
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	runnableInDB, errGetRunnable := integration.GetRunnable(request.Context(), details.UID, h.DB)
	if util.InternalHandlerError("integration.GetRunnable(int)", response, request, errGetRunnable) {
		return
	}
	if runnableInDB.OwnerID != usr.UID {
		http.Error(response, "You do not own this integration.", 403)
		return
	}

	err = integration.SaveCode(request.Context(), details.UID, details.Code, h.DB)
	if util.InternalHandlerError("integration.SaveCode(UID, code)", response, request, err) {
		return
	}
}

// HandleRun handles web requests to run a runnable.
func (h *IntegrationHandler) HandleRun(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
		return
	}

	var runnableUID int
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&runnableUID)
	if util.InternalHandlerError("json.Decode(int)", response, request, err) {
		return
	}

	runnableInDB, errGetRunnable := integration.GetRunnable(request.Context(), runnableUID, h.DB)
	if util.InternalHandlerError("integration.GetRunnable(int)", response, request, errGetRunnable) {
		return
	}
	if runnableInDB.OwnerID != usr.UID {
		http.Error(response, "You do not own this integration.", 403)
		return
	}

	runID, err := integrationState.Start(runnableUID, &integrationState.StartContext{
		TriggerKind: "manual",
		TriggerUID:  0,
	}, otto.New())
	if util.InternalHandlerError("integration.Start(runnable)", response, request, err) {
		return
	}

	b, err := json.Marshal(struct{ RunID string }{RunID: runID})
	if util.InternalHandlerError("json.Marshal(runID)", response, request, err) {
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// HandleEditRunnable handles web requests to edit a runnable.
func (h *IntegrationHandler) HandleEditRunnable(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
		return
	}

	var runnable integration.Runnable
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&runnable)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}
	runnable.OwnerID = usr.UID

	runnableInDB, errGetRunnable := integration.GetRunnable(request.Context(), runnable.UID, h.DB)
	if util.InternalHandlerError("integration.GetRunnable(int)", response, request, errGetRunnable) {
		return
	}
	if runnableInDB.OwnerID != usr.UID {
		http.Error(response, "You do not own this integration.", 403)
		return
	}

	// check all given Triggers have either no UID, or a UID belonging to a trigger which belongs to this user.
	for _, t := range runnable.Triggers {
		t.ParentUID = runnable.UID
		t.OwnerUID = usr.UID
		if t.UID == 0 {
			continue //we dont care about new columns
		}
		triggerInDB, errTrigger := integration.GetTriggerByUID(request.Context(), t.UID, h.DB)
		if util.InternalHandlerError("integration.GetTriggerByUID(struct)", response, request, errTrigger) {
			return
		}
		if triggerInDB.OwnerUID != usr.UID {
			http.Error(response, "You do not own this trigger.", 403)
			return
		}
	}

	err = integration.DoEditRunnable(request.Context(), &runnable, h.DB)
	if util.InternalHandlerError("integration.DoEditRunnable(struct)", response, request, err) {
		return
	}

	err = integrationState.RunnableChanged(&runnable)
	if util.InternalHandlerError("integrationState.RunnableChanged(runnable)", response, request, err) {
		return
	}
}

// HandleDeleteRunnable handles web requests to delete a runnable.
func (h *IntegrationHandler) HandleDeleteRunnable(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
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
		err = integrationState.RunnableChanged(&integration.Runnable{UID: id})
		if util.InternalHandlerError("integrationState.RunnableChanged(runnable)", response, request, err) {
			return
		}
	}
}

// HandleGetMine handles web requests to retrieve the integrations owned by an account.
func (h *IntegrationHandler) HandleGetMine(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
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

// HandleGetRuns handles web requests to retrieve a list of runIDs for a given runnableID.
func (h *IntegrationHandler) HandleGetRuns(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
		return
	}

	var IDs []int
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&IDs)
	if util.InternalHandlerError("json.Decode([]int)", response, request, err) {
		return
	}
	if len(IDs) != 1 {
		http.Error(response, "Can only handle a single runID", 500)
		return
	}
	id := IDs[0]

	runnable, errGetRunnable := integration.GetRunnable(request.Context(), id, h.DB)
	if util.InternalHandlerError("integration.GetRunnable(int)", response, request, errGetRunnable) {
		return
	}
	if runnable.OwnerID != usr.UID {
		http.Error(response, "You do not own this integration.", 403)
		return
	}

	runs, err := integration.GetRecentRunsForRunnable(request.Context(), id, time.Now().Add(-maxAgeRuns), h.DB)
	if util.InternalHandlerError("integration.GetRecentRunsForRunnable()", response, request, err) {
		return
	}

	b, err := json.Marshal(runs)
	if util.InternalHandlerError("json.Marshal(runs)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// HandleGetLogs handles web requests to retrieve log entries
func (h *IntegrationHandler) HandleGetLogs(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Integrations || !s.AccessWeb {
		http.Error(response, "You do not have permission to use integrations", 403)
		return
	}

	var filter struct {
		RunnableUID int
		RunID       string
		Offset      int
		Limit       int
		Info        bool
		Problem     bool
		Sys         bool
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&filter)
	if util.InternalHandlerError("json.Decode([]int)", response, request, err) {
		return
	}

	runnable, errGetRunnable := integration.GetRunnable(request.Context(), filter.RunnableUID, h.DB)
	if util.InternalHandlerError("integration.GetRunnable(int)", response, request, errGetRunnable) {
		return
	}
	if runnable.OwnerID != usr.UID {
		http.Error(response, "You do not own this integration.", 403)
		return
	}

	var logs []*integration.Log

	if filter.RunID != "" {
		logs, err = integration.GetLogsFilteredByRunnable(request.Context(), filter.RunnableUID, time.Now().Add(-maxAgeLogs), filter.RunID, filter.Offset, filter.Limit,
			filter.Info, filter.Problem, filter.Sys, h.DB)
	} else {
		logs, err = integration.GetLogsForRunnable(request.Context(), filter.RunnableUID, time.Now().Add(-maxAgeLogs), filter.Offset, filter.Limit,
			filter.Info, filter.Problem, filter.Sys, h.DB)
	}
	if util.InternalHandlerError("integration.GetLogsFilteredByRunnable()", response, request, err) {
		return
	}

	b, err := json.Marshal(logs)
	if util.InternalHandlerError("json.Marshal(logs)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
