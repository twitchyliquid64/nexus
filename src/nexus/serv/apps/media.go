package apps

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"nexus/data/fs"
	"nexus/data/session"
	"nexus/data/user"
	intfs "nexus/fs"
	"nexus/serv/util"
	"os"
	"path"
	"strings"
	"time"
)

// MediaApp represents the media player application.
type MediaApp struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (a *MediaApp) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	templatePath := ctx.Value("templatePath")
	if templatePath != nil {
		a.TemplatePath = templatePath.(string)
	} else {
		var err error
		a.TemplatePath, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	a.DB = db

	mux.HandleFunc("/app/media", a.Render)
	mux.HandleFunc("/app/media/vid", a.RenderVideoPlayer)
	mux.HandleFunc("/app/media/sources", a.DataSources)
	mux.HandleFunc("/app/media/files", a.Files)
	mux.HandleFunc("/app/media/getURL", a.GetURL)
	return nil
}

// Render generates page content.
func (a *MediaApp) Render(response http.ResponseWriter, request *http.Request) {
	_, _, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	util.LogIfErr("MediaApp.Render(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/media/main.html"), nil, response))
}

// RenderVideoPlayer generates page content.
func (a *MediaApp) RenderVideoPlayer(response http.ResponseWriter, request *http.Request) {
	_, _, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	util.LogIfErr("MediaApp.Render(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/media/videoplayer.html"), nil, response))
}

// DataSources handles JSON requests to list available sources.
func (a *MediaApp) DataSources(response http.ResponseWriter, request *http.Request) {
	_, u, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	sources, err := fs.GetSourcesForUser(request.Context(), u.UID, a.DB)
	if err != nil {
		log.Printf("fs.GetSourcesForUser() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	for i := range sources {
		sources[i].Value3 = ""
		sources[i].Value2 = ""
		sources[i].Value1 = ""
	}
	response.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(response).Encode(sources)
	if err != nil {
		http.Error(response, "Internal server error", 500)
		return
	}
}

// Files handles JSON requests to list files in a path.
func (a *MediaApp) Files(response http.ResponseWriter, request *http.Request) {
	_, u, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	var input struct {
		Path string `json:"path"`
	}
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		log.Printf("json.Decode() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	files, err := intfs.List(request.Context(), input.Path, u.UID)
	if err != nil {
		log.Printf("fs.List() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(response).Encode(files)
	if err != nil {
		http.Error(response, "Internal server error", 500)
		return
	}
}

// GetURL handles JSON requests to get the signed URL of a file.
func (a *MediaApp) GetURL(response http.ResponseWriter, request *http.Request) {
	_, u, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	var input struct {
		Path  string `json:"path"`
		Video bool   `json:"video"`
	}
	err = json.NewDecoder(request.Body).Decode(&input)
	if err != nil {
		log.Printf("json.Decode() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}

	fsSrc, err := intfs.SourceForPath(request.Context(), input.Path, u.UID)
	if err != nil {
		log.Printf("fs.List() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return
	}
	src, _ := intfs.ExpandSource(fsSrc)

	s3src, ok := src.(*intfs.S3)
	if !ok {
		http.Error(response, "Data source is not S3", 500)
		return
	}

	exp := time.Now().Add(time.Hour)
	if input.Video {
		exp = exp.Add(2 * time.Hour)
	}

	url := s3src.SignedURL(request.Context(), strings.TrimPrefix(input.Path, "/"+fsSrc.Prefix+"/"), exp, u.UID)
	response.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(response).Encode(map[string]string{"url": url})
	if err != nil {
		http.Error(response, "Internal server error", 500)
		return
	}
}

// EntryURL implements app.
func (a *MediaApp) EntryURL() string {
	return "/app/media"
}

// Name implements app.
func (a *MediaApp) Name() string {
	return "Media player"
}

// Icon implements app.
func (a *MediaApp) Icon() string {
	return "video_library"
}

// ShouldShowIcon implements app.
func (a *MediaApp) ShouldShowIcon(ctx context.Context, uid int) (bool, error) {
	attrs, err := user.GetAttrForUser(ctx, uid, a.DB)
	if err != nil {
		return false, err
	}
	for _, attr := range attrs {
		if strings.ToLower(attr.Name) == "media_player_icon" {
			if strings.Contains(strings.ToLower(attr.Val), "no") {
				return false, nil
			}
		}
	}
	return true, nil
}
