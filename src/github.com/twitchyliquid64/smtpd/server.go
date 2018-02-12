package smtpd

import (
	"context"
	"log"
	"net"
	"sync"
	"time"
)

// Server holds the configuration and state of our SMTP server
type Server struct {
	// Configuration
	listenAddr      string
	domain          string
	maxRecips       int
	maxIdleSeconds  int
	maxMessageBytes int

	// Dependencies
	dataStore      DataStore // Mailbox/message store
	globalShutdown chan bool // Shuts down Inbucket

	// State
	listener  net.Listener    // Incoming network connections
	waitgroup *sync.WaitGroup // Waitgroup tracks individual sessions
}

// NewServer creates a new Server instance with the specificed config
func NewServer(listenAddr, domain string, globalShutdown chan bool, ds DataStore) *Server {
	return &Server{
		listenAddr:      listenAddr,
		domain:          domain,
		maxRecips:       50,
		maxIdleSeconds:  600,
		maxMessageBytes: 1024 * 1024 * 4,
		globalShutdown:  globalShutdown,
		dataStore:       ds,
		waitgroup:       new(sync.WaitGroup),
	}
}

// Start the listener and handle incoming connections
func (s *Server) Start(ctx context.Context) error {
	addr, err := net.ResolveTCPAddr("tcp4", s.listenAddr)
	if err != nil {
		s.emergencyShutdown()
		return err
	}

	log.Printf("SMTP listening on TCP4 %v", addr)
	s.listener, err = net.ListenTCP("tcp4", addr)
	if err != nil {
		s.emergencyShutdown()
		return err
	}
	// Listener go routine
	go s.serve(ctx)

	// Wait for shutdown
	select {
	case <-ctx.Done():
		log.Printf("SMTP shutdown requested, connections will be drained")
	}

	// Closing the listener will cause the serve() go routine to exit
	return s.listener.Close()
}

// serve is the listen/accept loop
func (s *Server) serve(ctx context.Context) {
	// Handle incoming connections
	var tempDelay time.Duration
	for sessionID := 1; ; sessionID++ {
		if conn, err := s.listener.Accept(); err != nil {
			// There was an error accepting the connection
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				// Temporary error, sleep for a bit and try again
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("SMTP accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			} else {
				// Permanent error
				select {
				case <-ctx.Done():
					// SMTP is shutting down
					return
				default:
					// Something went wrong
					s.emergencyShutdown()
					return
				}
			}
		} else {
			tempDelay = 0
			s.waitgroup.Add(1)
			go s.startSession(sessionID, conn)
		}
	}
}

func (s *Server) emergencyShutdown() {
	select {
	case _ = <-s.globalShutdown:
	default:
		close(s.globalShutdown)
	}
}

// Drain causes the caller to block until all active SMTP sessions have finished
func (s *Server) Drain() {
	// Wait for sessions to close
	s.waitgroup.Wait()
}
