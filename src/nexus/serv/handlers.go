package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"nexus/serv/handlers"
	"reflect"
)

var httpHandlers = []handler{
	&handlers.CoreHandler{},
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

	return mux
}
