package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"nexus/data"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func die(msg string) {
	log.Fatal(msg)
}

var run = true
var dbFlag = flag.String("db", "dev.db", "path to the database file")
var listenerFlag = flag.String("listener", "localhost:8080", "Address to listen on")

func main() {
	flag.Parse()
	ctx := context.Background()

	log.Printf("Opening db: %q", *dbFlag)
	db, err := data.Init(ctx, "ql", *dbFlag)
	if err != nil {
		die(err.Error())
	}
	defer db.Close()

	mux := makeMux(ctx, db)

	for run {
		var servMutex sync.WaitGroup
		serv := makeServer(ctx, mux, *listenerFlag)
		log.Println("Start listening on", *listenerFlag)

		go func() {
			servMutex.Add(1)
			err := serv.ListenAndServe()
			if err != http.ErrServerClosed {
				log.Fatal("ListenAndServe()", err)
			} else {
				log.Println("HTTP server entered shutdown")
			}
			servMutex.Done()
		}()

		sig := waitInterrupt()
		serv.Close() // serv.Shutdown(ctx)
		servMutex.Wait()
		if sig == syscall.SIGHUP {
			log.Println("Got SIGHUP, reloading")
		} else {
			db.Close()
			os.Exit(0)
		}
	}
}

func waitInterrupt() os.Signal {
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return <-sig
}

func makeServer(ctx context.Context, mux *http.ServeMux, listenAddr string) *http.Server {
	return &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}
}
