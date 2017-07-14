package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/fs"
	"nexus/serv/util"
	"os"
)

// FSHandler handles endpoints which represent filesystem operations.
type FSHandler struct {
	DB *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *FSHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {

	h.DB = db
	mux.HandleFunc("/web/v1/fs/list", h.ListHandler)
	mux.HandleFunc("/web/v1/fs/save", h.AddHandler)
	mux.HandleFunc("/web/v1/fs/delete", h.DeleteHandler)
	mux.HandleFunc("/web/v1/fs/newFolder", h.NewFolderHandler)
	mux.HandleFunc("/web/v1/fs/download/", h.DownloadHandler)
	return nil
}

func (h *FSHandler) error(response http.ResponseWriter, request *http.Request, reason string, err error, extra map[string]interface{}) {
	log.Printf("FSHandler Error - %s: %v", reason, err)
	if extra == nil {
		extra = map[string]interface{}{}
	}
	extra["success"] = false
	extra["reason"] = reason
	b, errMarshal := json.Marshal(extra)
	if errMarshal != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(200)
	response.Write(b)
}

// ListHandler handles requests to list directories
func (h *FSHandler) ListHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	files, err := fs.List(request.Context(), details.Path, usr.UID)
	if err != nil {
		switch err {
		case os.ErrNotExist:
			h.error(response, request, "File or folder does not exist", err, map[string]interface{}{"path": details.Path})
		default:
			h.error(response, request, "List() failed", err, nil)
		}
		return
	}

	b, errMarshal := json.Marshal(files)
	if errMarshal != nil {
		h.error(response, request, "Result marshal failed", errMarshal, nil)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// AddHandler handles requests to add files
func (h *FSHandler) AddHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	err = fs.Save(request.Context(), details.Path, usr.UID, []byte(""))
	if err != nil {
		h.error(response, request, "Add() failed", err, nil)
		return
	}
}

// DeleteHandler handles requests to delete files
func (h *FSHandler) DeleteHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	err = fs.Delete(request.Context(), details.Path, usr.UID)
	if err != nil {
		switch err {
		case fs.ErrHasFiles:
			h.error(response, request, "Cannot delete a folder which contains files", err, map[string]interface{}{"path": details.Path})
		default:
			h.error(response, request, "Delete() failed", err, nil)
		}
		return
	}
}

// NewFolderHandler handles requests to create folders
func (h *FSHandler) NewFolderHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	err = fs.NewFolder(request.Context(), details.Path, usr.UID)
	if err != nil {
		h.error(response, request, "NewFolder() failed", err, nil)
		return
	}
}

// DownloadHandler handles requests to download a file
func (h *FSHandler) DownloadHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	// TODO: Get metadata first to check if exists / get filename.
	// Dont set the headers if we get an error there.

	response.Header().Set("Content-Disposition", "attachment;")
	path := request.URL.Path[len("/web/v1/fs/download"):]
	err = fs.Contents(request.Context(), path, usr.UID, response)
	if err != nil {
		log.Printf("Path: %q", path)
		h.error(response, request, "Contents() failed", err, nil)
		return
	}
}
