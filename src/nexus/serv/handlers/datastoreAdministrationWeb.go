package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"nexus/data/datastore"
	"nexus/serv/util"
)

// DatastoreAdministrationHandler handles requests used in management of datastores.
type DatastoreAdministrationHandler struct {
	DB *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *DatastoreAdministrationHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db

	mux.HandleFunc("/web/v1/data/list", h.HandleListV1)
	mux.HandleFunc("/web/v1/data/new", h.HandleNewV1)
	return nil
}

// HandleNewV1 handles a HTTP request to create a new datastore. TODO: Check name/owner combo doesnt already exist.
func (h *DatastoreAdministrationHandler) HandleNewV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Datastore datastore.Datastore
		Cols      []datastore.Column
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}
	details.Datastore.OwnerID = usr.UID

	storeUID, err := datastore.MakeDatastore(request.Context(), &details.Datastore, h.DB)
	if util.InternalHandlerError("datastore.MakeDatastore()", response, request, err) {
		return
	}

	for _, col := range details.Cols {
		err = datastore.MakeColumn(request.Context(), storeUID, &col, h.DB)
		if util.InternalHandlerError("datastore.MakeColumn()", response, request, err) {
			return
		}
	}
}

// HandleListV1 handles a HTTP request to list all the datastores that user has access to.
func (h *DatastoreAdministrationHandler) HandleListV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	datastores, err := datastore.GetDatastores(request.Context(), usr.AdminPerms.Data, usr.UID, h.DB)
	if util.InternalHandlerError("datastore.GetDatastores()", response, request, err) {
		return
	}

	b, err := json.Marshal(datastores)
	if util.InternalHandlerError("json.Marshal([]*datastore.Datastore)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
