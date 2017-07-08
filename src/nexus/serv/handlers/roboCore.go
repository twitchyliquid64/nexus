package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"nexus/data/session"
	"nexus/data/user"
	"nexus/serv/util"
)

// RoboCoreHandler handles feature-critical HTTP endpoints for robot accounts
type RoboCoreHandler struct {
	DB *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *RoboCoreHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db
	mux.HandleFunc("/api/v1/auth", h.HandleAuth)
	return nil
}

// HandleAuth handles requests to generate a session for a robot account.
func (h *RoboCoreHandler) HandleAuth(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&data)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}

	u, _ := url.Parse("http://kek/path?user=" + url.QueryEscape(data.Username) + "&password=" + url.QueryEscape(data.Password))
	r := http.Request{URL: u}

	ok, _, err := util.CheckAuth(ctx, &r, h.DB)
	if util.InternalHandlerError("util.CheckAuth()", response, request, err) {
		return
	}

	if ok {
		log.Printf("Got correct basicpass credentials for %s, creating robot session", data.Username)
		usr, err := user.Get(ctx, data.Username, h.DB)
		if util.InternalHandlerError("user.Get()", response, request, err) {
			return
		}

		if !usr.IsRobot {
			log.Printf("User account %q attempted to login to robot interface, login denied", usr.Username)
			http.Error(response, "Access Denied", 403)
			return
		}

		sid, err := session.Create(ctx, usr.UID, false, true, session.AuthPass, h.DB)
		if util.InternalHandlerError("session.Create()", response, request, err) {
			return
		}
		response.Write([]byte(sid))
	} else {
		http.Error(response, "Access Denied", 403)
	}
}
