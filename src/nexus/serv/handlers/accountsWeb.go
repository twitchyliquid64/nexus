package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"nexus/data/user"
	"nexus/serv/util"
)

// AccountsWebHandler handles HTTP endpoints looking up and setting accounts information.
type AccountsWebHandler struct {
	DB *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *AccountsWebHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db

	mux.HandleFunc("/web/v1/accounts", h.HandleListAccountsV1)
	return nil
}

// HandleListAccountsV1 handles web requests to list all accounts in the system.
func (h *AccountsWebHandler) HandleListAccountsV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	accounts, err := user.GetAll(request.Context(), h.DB)
	if util.InternalHandlerError("user.GetAll()", response, request, err) {
		return
	}

	b, err := json.Marshal(accounts)
	if util.InternalHandlerError("json.Marshal([]*user.DAO)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
