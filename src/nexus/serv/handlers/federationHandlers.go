package handlers

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"nexus/data/user"
	"nexus/serv/util"
	"time"
)

// FederationHandler handles HTTP endpoints for systems integration.
type FederationHandler struct {
	Enabled  bool
	CertPath string
	CaCert   *x509.Certificate
	DB       *sql.DB
}

// BindMux registers HTTP handlers on the given mux.
func (h *FederationHandler) BindMux(ctx context.Context, mux *http.ServeMux, db *sql.DB) error {
	h.DB = db
	h.Enabled = flag.Lookup("federation-enabled").Value.String() == "true"
	h.CertPath = flag.Lookup("federation-cert").Value.String()

	if h.Enabled {
		pemBytes, err := ioutil.ReadFile(h.CertPath)
		if err != nil {
			return err
		}
		certDERBlock, _ := pem.Decode(pemBytes)
		if certDERBlock == nil {
			return errors.New("No certificate data read from PEM")
		}
		h.CaCert, err = x509.ParseCertificate(certDERBlock.Bytes)
		if err != nil {
			return err
		}
	}

	mux.HandleFunc("/federation/v1/accounts/users", h.HandleGetUsers)
	return nil
}

func (h *FederationHandler) checkCertificates(request *http.Request) error {
	for i, cert := range request.TLS.PeerCertificates {
		if cert.NotAfter.Before(time.Now()) || cert.NotBefore.After(time.Now()) {
			return fmt.Errorf("Expired certificate presented at index %d", i)
		}

		if err := cert.CheckSignatureFrom(h.CaCert); err != nil {
			return fmt.Errorf("Certificate error at index %d: %s", i, err)
		}
		return nil // signature is valid
	}
	return fmt.Errorf("No valid certificate out of %d presented certificates", len(request.TLS.PeerCertificates))
}

func (h *FederationHandler) shouldAllowFederationRequest(request *http.Request) bool {
	if !h.Enabled || request.TLS == nil {
		log.Printf("Not servicing federation request (%s): federation-enabled=%v tls-nil=%v", request.URL.Path, h.Enabled, request.TLS == nil)
		return false
	}

	certErr := h.checkCertificates(request)
	if certErr != nil {
		log.Printf("Cert error in federation request (%s): %s", request.URL.Path, certErr.Error())
		return false
	}
	return true
}

// HandleGetUsers handles web requests to retrieve all users in the system.
func (h *FederationHandler) HandleGetUsers(response http.ResponseWriter, request *http.Request) {
	if !h.shouldAllowFederationRequest(request) {
		http.Error(response, "Internal server error", 500)
		return
	}

	accounts, err := user.GetAll(request.Context(), h.DB)
	if util.InternalHandlerError("user.GetAll()", response, request, err) {
		return
	}
	for i := range accounts {
		accounts[i].Attributes, err = user.GetAttrForUser(request.Context(), accounts[i].UID, h.DB)
		if util.InternalHandlerError("user.GetAttrForUser()", response, request, err) {
			return
		}
	}

	b, err := json.Marshal(accounts)
	if util.InternalHandlerError("json.Marshal([]*user.DAO)", response, request, err) {
		return
	}
	response.Header().Set("Content-Type", "application/json")
	response.Write(b)
}
