package apps

// NOTE:
// Make sure you have the offical youtube-dl installed from their website.
// Also need ffmpeg

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"nexus/data/session"
	"nexus/data/user"
	"nexus/fs"
	"nexus/serv/util"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/knadh/go-get-youtube/youtube"
)

type pendingDownload struct {
	VID          string `json:"vid"`
	DownloadPath string `json:"download_path"`
	UID          int
}

var (
	pendingDownloads    []pendingDownload
	pendingDownloadLock sync.Mutex
	downloadInProgress  bool
)

// YtdlApp represents the yt download to mp3 application.
type YtdlApp struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (a *YtdlApp) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
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

	mux.HandleFunc("/app/ytdl", a.render)
	mux.HandleFunc("/app/ytdl/metadata", a.serveMetadataInfo)
	mux.HandleFunc("/app/ytdl/queue", a.doEnqueueVideo)
	mux.HandleFunc("/app/ytdl/status", a.serveStatus)
	go a.downloaderRoutine()
	return nil
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *YtdlApp) downloaderRoutine() {
	var download pendingDownload
	tempDir, _ := ioutil.TempDir("", "ytdl")

	for {
		downloadInProgress = false
		time.Sleep(time.Second * 3)

		pendingDownloadLock.Lock()
		if len(pendingDownloads) > 0 {
			download, pendingDownloads = pendingDownloads[0], pendingDownloads[1:]
			pendingDownloadLock.Unlock()

			downloadInProgress = true
			downloadErr := downloadToMp3(download.VID, tempDir)
			if downloadErr != nil {
				removeContents(tempDir)
				continue
			}
			d, err := ioutil.ReadFile(path.Join(tempDir, "output.mp3"))
			if err != nil {
				log.Printf("ReadFile() err: %v", err)
				removeContents(tempDir)
				continue
			}
			err = fs.Save(context.Background(), download.DownloadPath, download.UID, d)
			if err != nil {
				log.Printf("fs.Save() err: %v", err)
			}
			removeContents(tempDir)
		} else {
			pendingDownloadLock.Unlock()
		}
	}
}

// actually downloads a VID to a dir. Transocodes to MP3 if necessary.
func downloadToMp3(vid, tempDir string) error {
	cmd := exec.Command("youtube-dl", "--extract-audio", "--audio-format", "mp3", "-o", path.Join(tempDir, "output.mp3"), "https://www.youtube.com/watch?v="+vid)
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Printf("youtube-dl err: %v", err)
		return err
	}

	files, err := ioutil.ReadDir(tempDir)
	if err != nil {
		log.Printf("ioutil.ReadDir(%q) err: %v", tempDir, err)
		return err
	}
	if len(files) != 1 {
		log.Printf("YtdlApp.downloaderRoutine(): Unexpected files after download: %+v", files)
		return err
	}
	if !strings.HasSuffix(files[0].Name(), ".mp3") {
		return errors.New("expected an mp3 file")
	}
	return nil
}

// returns true if user is authorized.
func (a *YtdlApp) handleCheckAuthorized(response http.ResponseWriter, request *http.Request) bool {
	_, u, err := util.AuthInfo(request, a.DB)
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return false
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return false
	}

	authorized, err := a.ShouldShowIcon(request.Context(), u.UID)
	if err != nil {
		log.Printf("YtdlApp.ShouldShowIcon() Error: %v", err)
		http.Error(response, "Internal server error", 500)
		return false
	}
	if !authorized {
		http.Error(response, "Unauthorized", 403)
		return false
	}
	return true
}

func (a *YtdlApp) doEnqueueVideo(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}
	var input pendingDownload
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&input)
	if util.InternalHandlerError("json.Decode(struct)", response, request, err) {
		return
	}
	_, u, err := util.AuthInfo(request, a.DB)
	if util.InternalHandlerError("util.AuthInfo)", response, request, err) {
		return
	}
	input.UID = u.UID

	pendingDownloadLock.Lock()
	pendingDownloads = append(pendingDownloads, input)
	defer pendingDownloadLock.Unlock()

	response.Header().Set("Content-Type", "application/json")
	response.Write([]byte("{\"success\": true}"))
}

func (a *YtdlApp) serveStatus(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	pendingDownloadLock.Lock()
	b, err := json.Marshal(struct {
		Idle       bool `json:"idle"`
		NumInQueue int  `json:"queue"`
	}{
		Idle:       !downloadInProgress,
		NumInQueue: len(pendingDownloads),
	})
	pendingDownloadLock.Unlock()
	if err != nil {
		http.Error(response, err.Error(), 500)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

func (a *YtdlApp) serveMetadataInfo(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	video, err := youtube.Get(request.FormValue("id"))
	if err != nil {
		log.Printf("YtdlApp.serveMetadataInfo() Error: %v", err)
		http.Error(response, err.Error(), 500)
		return
	}

	b, err := json.Marshal(video)
	if err != nil {
		http.Error(response, err.Error(), 500)
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}

func (a *YtdlApp) render(response http.ResponseWriter, request *http.Request) {
	if !a.handleCheckAuthorized(response, request) {
		return
	}

	util.ApplyStrictTransportSecurity(request, response)
	util.LogIfErr("YtdlApp.Render(): %v", util.RenderPage(path.Join(a.TemplatePath, "templates/apps/ytdl/index.html"), nil, response))
}

// EntryURL implements app.
func (a *YtdlApp) EntryURL() string {
	return "/app/ytdl"
}

// Name implements app.
func (a *YtdlApp) Name() string {
	return "Youtube DL"
}

// Icon implements app.
func (a *YtdlApp) Icon() string {
	return "video_call"
}

// ShouldShowIcon implements app.
func (a *YtdlApp) ShouldShowIcon(ctx context.Context, uid int) (bool, error) {
	attrs, err := user.GetAttrForUser(ctx, uid, a.DB)
	if err != nil {
		return false, err
	}
	for _, attr := range attrs {
		if strings.ToLower(attr.Name) == "ytdl" {
			if strings.Contains(strings.ToLower(attr.Val), "no") || strings.Contains(strings.ToLower(attr.Val), "den") { //no or deny or denied or no access
				return false, nil
			}
			return true, nil
		}
	}
	return false, nil
}
