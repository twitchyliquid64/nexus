package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"nexus/data"
	"nexus/fs"
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
var backupDuration = flag.Duration("backup-duration", 0, "(optional) Time between full backup runs")
var federationCert = flag.String("federation-cert", "", "Path to certificate to use to verify federation requests")
var federationEnabled = flag.Bool("federation-enabled", false, "Whether federation requests will be serviced or ignored")
var letsEncryptManager letsencrypt.Manager

func main() {
	flag.Parse()
	ctx := context.Background()

	log.Printf("Opening db: %q", *dbFlag)
	db, err := data.Init(ctx, *dbFlag)
	if err != nil {
		die(err.Error())
	}
	defer db.Close()

	if *backupDuration > 0 {
		data.StartBackups(*backupDuration)
	}

	err = messaging.Init(ctx, db)
	if err != nil {
		die(err.Error())
	}
	defer messaging.Deinit()

	err = fs.Initialize(ctx, db)
	if err != nil {
		die(err.Error())
	}

	err = integration.Initialise(ctx, db)
	if err != nil {
		log.Println("a")
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
