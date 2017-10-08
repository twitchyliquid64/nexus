package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/integration"
	"nexus/serv/handlers"
	"reflect"
	"strings"
)

var httpHandlers = []handler{
	&handlers.CoreHandler{},
	&handlers.AccountsWebHandler{},
	&handlers.DatastoreHandler{},
	&handlers.MessengerHandler{},
	&handlers.RoboCoreHandler{},
	&handlers.IntegrationHandler{},
	&handlers.MaintenanceHandler{},
	&handlers.FSHandler{},
	&handlers.SettingsHandler{},
	&handlers.DashboardHandler{},
	&handlers.FederationHandler{},
	&appsInternalHandler{},
}

type handler interface {
	BindMux(context.Context, *http.ServeMux, *sql.DB) error
}

func makeMux(ctx context.Context, db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	fs := http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	mux.Handle("/static/", fs)

	for _, handler := range httpHandlers {
		if err := handler.BindMux(ctx, mux, db); err != nil {
			log.Printf("[E] Cannot bind handler: %s", err.Error())
		} else {
			name := reflect.TypeOf(handler).String()
			name = name[strings.Index(name, ".")+1:]
			log.Printf("[mux] Registered %s", name)
		}
	}

	for _, appHandler := range apps {
		if err := appHandler.BindMux(ctx, mux, db); err != nil {
			log.Printf("[E] Cannot bind application %q: %s", appHandler.Name(), err.Error())
		} else {
			log.Printf("[mux] Initialized app %s", appHandler.Name())
		}
	}

	mux.Handle("/integration/", integration.WebTrigger)
	return mux
}
