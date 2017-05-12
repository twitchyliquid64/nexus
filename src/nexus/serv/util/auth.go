package util

import (
	"database/sql"
	"net/http"
	"nexus/data/session"
)

// AuthInfo returns the session, username, displayname of the logged-in user.
func AuthInfo(r *http.Request, db *sql.DB) (*session.DAO, string, string, error) {
	sidCookie, err := r.Cookie("sid")
	if err != nil {
		return nil, "", "", err
	}

	session, err := session.Get(r.Context(), sidCookie.Value, db)
	if err != nil {
		return nil, "", "", err
	}

	return session, "", "", nil
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
