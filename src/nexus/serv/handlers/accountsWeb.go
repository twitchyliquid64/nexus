package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data/datastore"
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
	mux.HandleFunc("/web/v1/account/edit", h.HandleEditAccountV1)
	mux.HandleFunc("/web/v1/account/create", h.HandleCreateAccountV1)
	mux.HandleFunc("/web/v1/account/delete", h.HandleDeleteAccountV1)
	mux.HandleFunc("/web/v1/account/setbasicpass", h.HandleSetBasicPassV1)
	return nil
}

// HandleSetBasicPassV1 handles web requests to set the basic password of an account.
func (h *AccountsWebHandler) HandleSetBasicPassV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var details struct {
		UID  int
		Pass string
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	candidateUser, err := user.GetByUID(request.Context(), details.UID, h.DB)
	if util.InternalHandlerError("user.GetByUID()", response, request, err) {
		return
	}

	if err := user.SetAuth(request.Context(), candidateUser.UID, details.Pass, candidateUser.AdminPerms.Accounts, candidateUser.AdminPerms.Data, candidateUser.AdminPerms.Integrations, h.DB); err != nil {
		http.Error(response, "Database error", 500)
		log.Printf("Error when change auth of %+v: %s", candidateUser, err)
	}
}

// HandleDeleteAccountV1 handles web requests to delete accounts.
func (h *AccountsWebHandler) HandleDeleteAccountV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var uids []int
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&uids)
	if util.InternalHandlerError("json.Decode([]int])", response, request, err) {
		return
	}
	for _, uid := range uids {
		if err := user.Delete(request.Context(), uid, h.DB); err != nil {
			http.Error(response, "Database error", 500)
			log.Printf("Error when deleting UID %d: %s", uid, err)
		}
	}
}

// HandleCreateAccountV1 handles web requests to create an account.
func (h *AccountsWebHandler) HandleCreateAccountV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var newUserObject user.DAO
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&newUserObject)
	if util.InternalHandlerError("json.Decode(user.DAO)", response, request, err) {
		return
	}
	if err := user.Create(request.Context(), &newUserObject, h.DB); err != nil {
		http.Error(response, "Database error", 500)
		log.Printf("Error when creating user %+v: %s", newUserObject, err)
	}
}

// HandleEditAccountV1 handles web requests to edit an account.
func (h *AccountsWebHandler) HandleEditAccountV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var newUserObject user.DAO
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&newUserObject)
	if util.InternalHandlerError("json.Decode(user.DAO)", response, request, err) {
		return
	}
	if err := user.Update(request.Context(), &newUserObject, h.DB); err != nil {
		http.Error(response, "Database error", 500)
		log.Printf("Error when updating user %d(%s): %s", newUserObject.UID, newUserObject.Username, err)
	}
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

	for i := range accounts {
		accounts[i].Grants, err = datastore.ListByUser(request.Context(), accounts[i].UID, h.DB)
		if util.InternalHandlerError("datastore.ListByUser()", response, request, err) {
			return
		}
	}

	b, err := json.Marshal(accounts)
	if util.InternalHandlerError("json.Marshal([]*user.DAO)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
