package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	dfs "nexus/data/fs"
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
	mux.HandleFunc("/web/v1/fs/save", h.SaveHandler)
	mux.HandleFunc("/web/v1/fs/delete", h.DeleteHandler)
	mux.HandleFunc("/web/v1/fs/newFolder", h.NewFolderHandler)
	mux.HandleFunc("/web/v1/fs/download/", h.DownloadHandler)
	mux.HandleFunc("/web/v1/fs/upload", h.UploadHandler)
	mux.HandleFunc("/web/v1/fs/actions", h.GetActionsHandler)
	mux.HandleFunc("/web/v1/fs/runAction", h.RunActionHandler)
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

// UploadHandler handles requests to upload a single file
func (h *FSHandler) UploadHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	request.ParseMultipartForm(1024 * 1024)
	file, handler, err := request.FormFile("upload")
	if err != nil {
		h.error(response, request, "Upload read failed", err, nil)
		return
	}
	defer file.Close()

	//log.Printf("Got upload to base %s with filename %s for %s", request.FormValue("path"), handler.Filename, usr.DisplayName)
	err = fs.Upload(request.Context(), request.FormValue("path")+"/"+handler.Filename, usr.UID, file)
	if err != nil {
		h.error(response, request, "Upload failed", err, nil)
		return
	}
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

// SaveHandler handles requests to save files
func (h *FSHandler) SaveHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Path string `json:"path"`
		Data string `json:"data"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	err = fs.Save(request.Context(), details.Path, usr.UID, []byte(details.Data))
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
		Path     string `json:"path"`
		IsFolder bool   `json:"isFolder"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	// S3 directories have a trailing slash on them.
	if src, err2 := fs.SourceForPath(request.Context(), details.Path, usr.UID); err2 == nil && src.Kind == dfs.FSSourceS3 {
		if details.IsFolder {
			details.Path = details.Path + "/"
		}
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

	if request.URL.Query().Get("composition") != "inline" {
		response.Header().Set("Content-Disposition", "attachment;")
	}

	path := request.URL.Path[len("/web/v1/fs/download"):]
	err = fs.Contents(request.Context(), path, usr.UID, response)
	if err != nil {
		log.Printf("Path: %q", path)
		h.error(response, request, "Contents() failed", err, nil)
		return
	}
}

// GetActionsHandler handles requests to list the actions available on a file.
func (h *FSHandler) GetActionsHandler(response http.ResponseWriter, request *http.Request) {
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

	actions, err := fs.Actions(request.Context(), details.Path, usr.UID)
	if err != nil {
		h.error(response, request, "Actions() failed", err, nil)
		return
	}

	b, errMarshal := json.Marshal(actions)
	if errMarshal != nil {
		h.error(response, request, "Result marshal failed", errMarshal, nil)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

// RunActionHandler handles requests to run an action on a path.
func (h *FSHandler) RunActionHandler(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	var details struct {
		Path    string            `json:"path"`
		ID      string            `json:"id"`
		Payload map[string]string `json:"payload"`
	}
	decoder := json.NewDecoder(request.Body)
	err = decoder.Decode(&details)
	if err != nil {
		h.error(response, request, "Request decode failed", err, nil)
		return
	}

	results, err := fs.RunAction(request.Context(), details.Path, usr.UID, details.ID, details.Payload)
	if err != nil {
		h.error(response, request, err.Error(), err, nil)
		return
	}

	b, errMarshal := json.Marshal(results)
	if errMarshal != nil {
		h.error(response, request, "Result marshal failed", errMarshal, nil)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
