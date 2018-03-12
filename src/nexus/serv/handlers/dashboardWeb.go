package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"nexus/fs"
	"nexus/serv/util"
	"os"
	"path"
	"sort"
	"strings"
)

// DashboardHandler handles endpoints which are used to get the dashboard content
type DashboardHandler struct {
	TemplatePath string
	DB           *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *DashboardHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	templatePath := ctx.Value("templatePath")
	if templatePath != nil {
		h.TemplatePath = templatePath.(string)
	} else {
		var err error
		h.TemplatePath, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	h.DB = db

	mux.HandleFunc("/dashboard/main", h.Render)
	return nil
}

type cardConfig struct {
	Color   string `json:"color"`
	Icon    string `json:"icon"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Subtext string `json:"subtext"`
	Width   int    `json:"width"`
}

type logConfig struct {
	Icon          string  `json:"icon"`
	IconColor     string  `json:"icon-color"`
	Title         string  `json:"title"`
	Subtitle      string  `json:"subtitle"`
	SecondaryIcon string  `json:"secondary-icon"`
	Fill          bool    `json:"fill"`
	Height        float64 `json:"height"`

	Items []struct {
		Type string `json:"type"`

		// type == `collection-item`
		Title     string `json:"title"`
		Icon      string `json:"icon"`
		IconColor string `json:"icon-color"`

		Sections []struct {
			Type  string `json:"type"`
			Class string `json:"class"`
			Text  string `json:"text"`
		} `json:"sections"`
	} `json:"items"`
}

type listConfig struct {
	Icon          string `json:"icon"`
	IconColor     string `json:"icon-color"`
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	SecondaryIcon string `json:"secondary-icon"`
	Fill          bool   `json:"fill"`
	Items         []struct {
		Title     string `json:"title"`
		Text      string `json:"text"`
		Tag       string `json:"tag"`
		TagColor  string `json:"tag-color"`
		Icon      string `json:"icon"`
		IconColor string `json:"icon-color"`

		ChartType    string                 `json:"chart-type"`
		ChartData    []int                  `json:"chart-data"`
		ChartOptions map[string]interface{} `json:"chart-options"`
	} `json:"items"`
}

type renderData struct {
	Cards []cardConfig
	Lists []listConfig
	Logs  []logConfig
}

type byName []fs.ListResultItem

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }

func contents(ctx context.Context, path string, userID int) ([]byte, error) {
	var out bytes.Buffer
	err := fs.Contents(ctx, "/minifs/"+path, userID, &out)
	return out.Bytes(), err
}

func loadRenderData(ctx context.Context, files []fs.ListResultItem, userID int) (*renderData, error) {
	out := &renderData{}
	sort.Sort(byName(files))

	for _, file := range files {
		s := strings.Index(file.Name, "-")
		if s < 0 || file.ItemKind != fs.KindFile {
			continue
		}

		switch file.Name[len("dashboard/"):s] {
		case "card":
			content, err := contents(ctx, file.Name, userID)
			if err != nil {
				return nil, err
			}
			c := cardConfig{Width: 3}
			err = json.Unmarshal(content, &c)
			if err != nil {
				log.Printf("Failed to unmarshal %q: %v", file.Name, err)
				continue
			}
			out.Cards = append(out.Cards, c)
		case "list":
			content, err := contents(ctx, file.Name, userID)
			if err != nil {
				return nil, err
			}
			c := listConfig{Icon: "settings"}
			err = json.Unmarshal(content, &c)
			if err != nil {
				log.Printf("Failed to unmarshal %q: %v", file.Name, err)
				continue
			}
			out.Lists = append(out.Lists, c)
		case "log":
			content, err := contents(ctx, file.Name, userID)
			if err != nil {
				return nil, err
			}
			c := logConfig{Icon: "settings"}
			err = json.Unmarshal(content, &c)
			if err != nil {
				log.Printf("Failed to unmarshal %q: %v", file.Name, err)
				continue
			}
			out.Logs = append(out.Logs, c)
		}
	}

	return out, nil
}

// Render is a HTTP handler which returns the current dashboard.
func (h *DashboardHandler) Render(response http.ResponseWriter, request *http.Request) {
	_, usr, err := util.AuthInfo(request, h.DB)
	if util.UnauthenticatedOrError(response, request, err) {
		return
	}

	configs, err := fs.List(request.Context(), "/minifs/dashboard", usr.UID)
	if err == os.ErrNotExist {
		return
	}
	if util.InternalHandlerError("fs.List()", response, request, err) {
		return
	}

	renderData, err := loadRenderData(request.Context(), configs, usr.UID)
	if util.InternalHandlerError("loadRenderData()", response, request, err) {
		return
	}

	t, err := template.New("t").Funcs(template.FuncMap{
		"chartData": func(data []int) string {
			out := ""
			for i, point := range data {
				out += fmt.Sprint(point)
				if i < len(data)-1 {
					out += ","
				}
			}
			return out
		},
		"logSize": func(height float64) string {
			return fmt.Sprintf("%dpx", int(height*64.1))
		}}).Delims("{!{", "}!}").ParseFiles(path.Join(h.TemplatePath, "templates", "dashboard.html"))

	if util.InternalHandlerError("template.Parse()", response, request, err) {
		return
	}

	util.ApplyStrictTransportSecurity(request, response)
	err = t.ExecuteTemplate(response, "dashboard.html", renderData)
	if util.InternalHandlerError("template.Execute()", response, request, err) {
		return
	}
}
