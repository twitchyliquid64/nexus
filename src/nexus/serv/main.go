package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"nexus/data"
	"nexus/integration"
	"nexus/messaging"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"rsc.io/letsencrypt"
)

func die(msg string) {
	log.Fatal(msg)
}

var dbFlag = flag.String("db", "dev.db", "path to the database file")
var listenerFlag = flag.String("listener", "localhost:8080", "Address to listen on")
var tlsCacheFileFlag = flag.String("key-cache", "", "Path to store SSL secrets")
var allowedDomains = flag.String("domains", "", "Comma-separated list of domains which are allowed to work")
var letsEncryptManager letsencrypt.Manager

func main() {
	flag.Parse()
	ctx := context.Background()

	log.Printf("Opening db: %q", *dbFlag)
	db, err := data.Init(ctx, "ql", *dbFlag)
	if err != nil {
		die(err.Error())
	}
	defer db.Close()

	err = messaging.Init(ctx, db)
	if err != nil {
		die(err.Error())
	}
	defer messaging.Deinit()

	err = integration.Initialise(ctx, db)
	if err != nil {
		die(err.Error())
	}

	// if we are doing the TLS thing, setup our state
	if *tlsCacheFileFlag != "" {
		if *allowedDomains != "" {
			letsEncryptManager.SetHosts(strings.Split(*allowedDomains, ","))
		}
		err = letsEncryptManager.CacheFile(*tlsCacheFileFlag)
		if err != nil {
			die(err.Error())
		}
	}

	mux := makeMux(ctx, db)
	for {
		runServ(ctx, mux, db)
	}
}

func runServ(ctx context.Context, mux *http.ServeMux, db *sql.DB) {
	var servMutex sync.WaitGroup
	serv := makeServer(ctx, mux, *listenerFlag)
	log.Println("Start listening on", *listenerFlag)

	go func() {
		var err error
		servMutex.Add(1)
		if *tlsCacheFileFlag == "" {
			err = serv.ListenAndServe()
		} else {
			err = serv.ListenAndServeTLS("", "")
		}

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
		messaging.Deinit()
		db.Close()
		// TODO: Gracefully close down integrations
		os.Exit(0)
	}
}

func waitInterrupt() os.Signal {
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	return <-sig
}

func makeServer(ctx context.Context, mux *http.ServeMux, listenAddr string) *http.Server {
	s := &http.Server{
		Addr:    listenAddr,
		Handler: mux,
	}
	if *tlsCacheFileFlag != "" {
		s.TLSConfig = &tls.Config{
			GetCertificate: letsEncryptManager.GetCertificate,
		}
	}
	return s
}
