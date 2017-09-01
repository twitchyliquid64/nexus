package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/integration"
	"nexus/serv/handlers"
	"reflect"
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
			log.Printf("[mux] Registered %s", reflect.TypeOf(handler).String()[len("*handlers."):])
		}
	}

	mux.Handle("/integration/", integration.WebTrigger)

	return mux
}
