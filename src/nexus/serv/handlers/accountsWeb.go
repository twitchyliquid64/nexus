package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"log"
	"net/http"
	"nexus/data/datastore"
	"nexus/data/user"
	"nexus/serv/util"
	"strconv"

	"github.com/pquerna/otp/totp"

	"golang.org/x/crypto/bcrypt"
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
	mux.HandleFunc("/web/v1/account/addgrant", h.HandleAddGrantV1)
	mux.HandleFunc("/web/v1/account/delgrant", h.HandleDeleteGrantV1)
	mux.HandleFunc("/web/v1/account/auths", h.HandleListAuthV1)
	mux.HandleFunc("/web/v1/account/addauth", h.HandleAddAuthV1)
	mux.HandleFunc("/web/v1/account/delauth", h.HandleDeleteAuthV1)
	mux.HandleFunc("/web/v1/genotp", h.GenOTPImg)
	return nil
}

// HandleSetBasicPassV1 handles web requests to set the basic password of an account.
func (h *AccountsWebHandler) HandleSetBasicPassV1(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
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
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
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

// HandleDeleteGrantV1 handles web requests to delete a datastore grant.
func (h *AccountsWebHandler) HandleDeleteGrantV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var input struct {
		Action string `json:"action"`
		GID    int    `json:"gid"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&input)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	err = datastore.DeleteGrant(request.Context(), input.GID, h.DB)
	if util.InternalHandlerError("datastore.DeleteGrant()", response, request, err) {
		return
	}
}

// HandleAddGrantV1 handles web requests to grant access to a datastore for a user.
func (h *AccountsWebHandler) HandleAddGrantV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var input struct {
		Action string `json:"action"`
		Dsuid  int    `json:"dsuid"`
		RW     bool   `json:"rw"`
		UID    int    `json:"uid"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&input)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	_, err = datastore.GetDatastore(request.Context(), input.Dsuid, h.DB)
	if util.InternalHandlerError("datastore.GetDatastore() - doesnt exist", response, request, err) {
		return
	}

	id, err := datastore.MakeGrant(request.Context(), &datastore.Grant{
		UsrUID:   input.UID,
		DsUID:    input.Dsuid,
		ReadOnly: input.RW,
	}, h.DB)
	if util.InternalHandlerError("datastore.MakeGrant()", response, request, err) {
		return
	}
	log.Printf("Grant ID=%d\n", id)
}

// HandleCreateAccountV1 handles web requests to create an account.
func (h *AccountsWebHandler) HandleCreateAccountV1(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
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
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
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

	seen := map[int]bool{}
	existingAttrs, err2 := user.GetAttrForUser(request.Context(), newUserObject.UID, h.DB)
	if util.InternalHandlerError("datastore.GetAttrForUser()", response, request, err2) {
		return
	}
	for _, a := range newUserObject.Attributes {
		a.UserID = newUserObject.UID
		if a.UID == 0 {
			if err := user.CreateAttr(request.Context(), a, h.DB); err != nil {
				http.Error(response, "Database error", 500)
				log.Printf("Error when creating Attr %s.%s: %s", a.KindStr(), a.Name, err.Error())
			}
		} else {
			seen[a.UID] = true
			if err := user.UpdateAttr(request.Context(), a, h.DB); err != nil {
				http.Error(response, "Database error", 500)
				log.Printf("Error when updating Attr %s.%s: %s", a.KindStr(), a.Name, err.Error())
			}
		}
	}
	for _, a := range existingAttrs {
		if !seen[a.UID] {
			if err := user.DeleteAttr(request.Context(), a.UID, h.DB); err != nil {
				http.Error(response, "Database error", 500)
				log.Printf("Error when deleting Attr %d: %s", a.UID, err.Error())
			}
		}
	}
}

// HandleListAccountsV1 handles web requests to list all accounts in the system.
func (h *AccountsWebHandler) HandleListAccountsV1(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
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
		accounts[i].Attributes, err = user.GetAttrForUser(request.Context(), accounts[i].UID, h.DB)
		if util.InternalHandlerError("user.GetAttrForUser()", response, request, err) {
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

// GenOTPImg generates an OTP QR code and secret.
func (h *AccountsWebHandler) GenOTPImg(response http.ResponseWriter, request *http.Request) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "nexus",
		AccountName: request.FormValue("account"),
	})
	if util.InternalHandlerError("totp.Generate()", response, request, err) {
		return
	}
	// Convert TOTP key into a QR code encoded as a PNG image.
	var buf bytes.Buffer
	img, err := key.Image(250, 250)
	if util.InternalHandlerError("key.Image()", response, request, err) {
		return
	}

	png.Encode(&buf, img)
	qrStr := base64.StdEncoding.EncodeToString(buf.Bytes())

	out := struct {
		Img string
		Key string
	}{Img: qrStr, Key: key.Secret()}

	b, err := json.Marshal(&out)
	if util.InternalHandlerError("json.Marshal(struct)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// HandleListAuthV1 handles web requests to list the auths for a user.
func (h *AccountsWebHandler) HandleListAuthV1(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var uid int
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&uid)
	if util.InternalHandlerError("json.Decode(int)", response, request, err) {
		return
	}

	auths, err := user.GetAuthForUser(request.Context(), uid, h.DB)
	if util.InternalHandlerError("user.GetAuthForUser()", response, request, err) {
		return
	}

	for i := range auths {
		auths[i].Val1 = "{REDACTED}" //lets not expose password hashes and OTPs to the frontend
	}

	b, err := json.Marshal(auths)
	if util.InternalHandlerError("json.Marshal([]*user.DAO)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// HandleAddAuthV1 handles web requests to add auth for a user
func (h *AccountsWebHandler) HandleAddAuthV1(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts {
		http.Error(response, "You do not have permission to manage accounts", 403)
		return
	}

	var input user.Auth
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&input)
	if util.InternalHandlerError("json.Decode(user.Auth)", response, request, err) {
		return
	}

	if input.Kind == user.KindPassword {
		b, err2 := bcrypt.GenerateFromPassword([]byte(input.Val1+"yoloSalty"+strconv.Itoa(input.UserID)), bcrypt.DefaultCost)
		if util.InternalHandlerError("bcrypt.GenerateFromPassword()", response, request, err2) {
			return
		}
		input.Val1 = hex.EncodeToString(b)
	}

	err = user.CreateAuth(request.Context(), &input, h.DB)
	if util.InternalHandlerError("user.CreateAuth()", response, request, err) {
		return
	}
}

// HandleDeleteAuthV1 handles web requests to remove an auth for a user
func (h *AccountsWebHandler) HandleDeleteAuthV1(response http.ResponseWriter, request *http.Request) {
	s, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	if !usr.AdminPerms.Accounts || !s.AccessWeb {
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
		if err := user.DeleteAuth(request.Context(), uid, h.DB); err != nil {
			http.Error(response, "Database error", 500)
			log.Printf("Error when deleting UID %d: %s", uid, err)
		}
	}
}
