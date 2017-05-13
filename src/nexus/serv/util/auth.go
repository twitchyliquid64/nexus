package util

import (
	"database/sql"
	"log"
	"net/http"
	"nexus/data/session"
	"nexus/data/user"
)

// AuthInfo returns the session, username, displayname of the logged-in user.
func AuthInfo(r *http.Request, db *sql.DB) (*session.DAO, *user.DAO, error) {
	sidCookie, err := r.Cookie("sid")
	if err != nil {
		return nil, nil, err
	}

	session, err := session.Get(r.Context(), sidCookie.Value, db)
	if err != nil {
		return nil, nil, err
	}

	usr, err := user.GetByUID(r.Context(), session.UID, db)
	if err != nil {
		return nil, nil, err
	}

	return session, usr, nil
}

func getCookieByName(cookie []*http.Cookie, name string) string {
	cookieLen := len(cookie)
	result := ""
	for i := 0; i < cookieLen; i++ {
		if cookie[i].Name == name {
			result = cookie[i].Value
		}
	}
	return result
}

// UnauthenticatedOrError sanely handles the logging and HTTP response for both errors and unauthenticated requests.
// Returns true if request handling should halt (error or unauth)
func UnauthenticatedOrError(response http.ResponseWriter, request *http.Request, err error) bool {
	if err == session.ErrInvalidSession || err == http.ErrNoCookie {
		http.Redirect(response, request, "/login", 303)
		return true
	} else if err != nil {
		log.Printf("AuthInfo() Error: %s", err)
		http.Error(response, "Internal server error", 500)
		return true
	}
	return false
}
