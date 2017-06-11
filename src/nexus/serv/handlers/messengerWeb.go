package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"nexus/data/messaging"
	"nexus/serv/util"
	"strconv"
)

// MessengerHandler handles HTTP endpoints for the 'messenger' IM tab.
type MessengerHandler struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *MessengerHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db

	mux.HandleFunc("/web/v1/messenger/conversations", h.HandleConversations)
	mux.HandleFunc("/web/v1/messenger/messages", h.HandleMessages)
	return nil
}

// HandleConversations handles web requests to retrieve the conversations which an account knows of.
func (h *MessengerHandler) HandleConversations(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	srcs, err := messaging.GetAllSourcesForUser(request.Context(), usr.UID, h.DB)
	if util.InternalHandlerError("messaging.GetAllSourcesForUser()", response, request, err) {
		return
	}

	convos, err := messaging.GetConversationsForUser(request.Context(), usr.UID, h.DB)
	if util.InternalHandlerError("messaging.GetConversationsForUser()", response, request, err) {
		return
	}

	outData := struct {
		Conversations []*messaging.Conversation
		Sources       []*messaging.Source
	}{
		Conversations: convos,
		Sources:       srcs,
	}

	b, err := json.Marshal(outData)
	if util.InternalHandlerError("json.Marshal(convos,sources)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// HandleMessages handles web requests to retrieve messages in a conversation.
func (h *MessengerHandler) HandleMessages(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	err = request.ParseForm()
	if util.InternalHandlerError("request.ParseForm()", response, request, err) {
		return
	}

	cID, err := strconv.Atoi(request.FormValue("cid"))
	if util.InternalHandlerError("strconv.Atoi(cid)", response, request, err) {
		return
	}

	convos, err := messaging.GetConversationsForUser(request.Context(), usr.UID, h.DB)
	if util.InternalHandlerError("messaging.GetConversationsForUser()", response, request, err) {
		return
	}
	foundConvo := false
	for _, convo := range convos { //check the requested cID is actually owned by the user requesting it
		if convo.UID == cID {
			foundConvo = true
			break
		}
	}
	if !foundConvo {
		http.Error(response, "You do not own this conversation.", 403)
		return
	}

	messages, err := messaging.GetMessagesForConversation(request.Context(), cID, h.DB)
	if util.InternalHandlerError("messaging.GetMessagesForConversation()", response, request, err) {
		return
	}

	b, err := json.Marshal(messages)
	if util.InternalHandlerError("json.Marshal([]messages)", response, request, err) {
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
