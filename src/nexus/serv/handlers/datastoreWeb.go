package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"nexus/data/datastore"
	"nexus/serv/util"
	"strconv"
)

//TODO: Rename to datastoreWeb

// DatastoreHandler handles requests used in management of datastores.
type DatastoreHandler struct {
	DB *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *DatastoreHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db

	mux.HandleFunc("/web/v1/data/list", h.HandleListV1)
	mux.HandleFunc("/web/v1/data/new", h.HandleNewV1)
	mux.HandleFunc("/web/v1/data/edit", h.HandleEditV1)
	mux.HandleFunc("/web/v1/data/delete", h.HandleDeleteV1)
	mux.HandleFunc("/web/v1/data/query", h.HandleQueryV1)
	mux.HandleFunc("/web/v1/data/insert", h.HandleInsertV1)
	return nil
}

// HandleNewV1 handles a HTTP request to create a new datastore. TODO: Check name/owner combo doesnt already exist.
func (h *DatastoreHandler) HandleNewV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details datastore.Datastore
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}
	details.OwnerID = usr.UID

	err = datastore.DoCreate(request.Context(), &details, h.DB)
	if util.InternalHandlerError("datastore.DoCreate()", response, request, err) {
		return
	}
}

// HandleDeleteV1 handles a HTTP request to delete a datastore.
func (h *DatastoreHandler) HandleDeleteV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var dUIDs []int
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&dUIDs)
	if util.InternalHandlerError("json.Decode([]int])", response, request, err) {
		return
	}

	for _, dUID := range dUIDs {
		storedDS, err := datastore.GetDatastore(request.Context(), dUID, h.DB)
		if util.InternalHandlerError("datastore.GetDatastore()", response, request, err) {
			return
		}
		if storedDS.OwnerID != usr.UID && !usr.AdminPerms.Data {
			http.Error(response, "You do not own this datastore.", 403)
			return
		}

		err = datastore.DoDelete(request.Context(), storedDS, h.DB)
		if util.InternalHandlerError("datastore.DoDelete()", response, request, err) {
			return
		}
	}
}

// HandleInsertV1 handles a HTTP request to insert into a datastore.
func (h *DatastoreHandler) HandleInsertV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if err := request.ParseForm(); err != nil {
		http.Error(response, "Could not parse form", 400)
		return
	}

	dsID, err := strconv.Atoi(request.FormValue("ds"))
	if util.InternalHandlerError("extractdsID-HandleInsertV1()", response, request, err) {
		return
	}
	storedDS, err := datastore.GetDatastore(request.Context(), dsID, h.DB)
	if util.InternalHandlerError("datastore.GetDatastore()", response, request, err) {
		return
	}
	if storedDS.OwnerID != usr.UID && !usr.AdminPerms.Data {
		http.Error(response, "You do not own this datastore.", 403)
		return
	}

	cols, err := util.ExtractColumnList(request.FormValue("cols"))
	if util.InternalHandlerError("util.ExtractColumnList()", response, request, err) {
		return
	}

	err = datastore.DoStreamingInsert(request.Context(), request.Body, dsID, cols, h.DB)
	util.InternalHandlerError("datastore.DoStreamingInsert()", response, request, err)
}

// HandleQueryV1 handles a HTTP request to query a datastore.
func (h *DatastoreHandler) HandleQueryV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var query datastore.Query
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&query)
	if util.InternalHandlerError("json.Decode(datastoreQuery)", response, request, err) {
		return
	}
	storedDS, err := datastore.GetDatastore(request.Context(), query.UID, h.DB)
	if util.InternalHandlerError("datastore.GetDatastore()", response, request, err) {
		return
	}
	if storedDS.OwnerID != usr.UID && !usr.AdminPerms.Data {
		http.Error(response, "You do not own this datastore.", 403)
		return
	}

	err = datastore.DoStreamingQuery(request.Context(), response, query, h.DB)
	if util.InternalHandlerError("datastore.DoStreamingQuery()", response, request, err) {
		return
	}

	return
}

// HandleEditV1 handles a HTTP request to edit a datastore.
func (h *DatastoreHandler) HandleEditV1(response http.ResponseWriter, request *http.Request) {
	_, _, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	http.Error(response, "Editing of datastores is not supported. Delete and re-create the datastore.", 501)
	return
}

// HandleListV1 handles a HTTP request to list all the datastores that user has access to.
func (h *DatastoreHandler) HandleListV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	datastores, err := datastore.GetDatastores(request.Context(), usr.AdminPerms.Data, usr.UID, h.DB)
	if util.InternalHandlerError("datastore.GetDatastores()", response, request, err) {
		return
	}

	for _, ds := range datastores {
		ds.Cols, err = datastore.GetColumns(request.Context(), ds.UID, h.DB)
		if util.InternalHandlerError("datastore.GetColumns()", response, request, err) {
			return
		}
	}

	b, err := json.Marshal(datastores)
	if util.InternalHandlerError("json.Marshal([]*datastore.Datastore)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
