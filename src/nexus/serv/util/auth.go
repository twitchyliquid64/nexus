package util

import (
	"context"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"nexus/data/session"
	"nexus/data/user"
	"strconv"

	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// AuthDetails contains semantic information about a successful or unsuccessful auth
type AuthDetails struct {
	BasicFallbackUsed bool
	OTPWanted         bool
	OTPUsed           bool
	PassUsed          bool
	PassedMethod      []string
}

// CheckAuth checks if the provided credentials are valid for that user.
func CheckAuth(ctx context.Context, request *http.Request, db *sql.DB) (bool, AuthDetails, error) {
	usr, err := user.Get(ctx, request.FormValue("user"), db)
	if err != nil {
		return false, AuthDetails{}, err
	}
	authMethods, err := user.GetAuthForUser(ctx, usr.UID, db)
	if err != nil {
		return false, AuthDetails{}, err
	}

	// Fallback to basic password auth if not auth methods are setup.
	if len(authMethods) == 0 {
		ok, _, err := user.CheckBasicAuth(ctx, request.FormValue("user"), request.FormValue("password"), db)
		if ok || err != nil {
			return ok, AuthDetails{BasicFallbackUsed: true}, err
		}
	}

	didPassOne := false
	var details AuthDetails
	for _, method := range authMethods {
		didPass := false
		methodString := ""

		// Do validity check specific to the type of auth (password or OTP etc)
		switch method.Kind {
		case user.KindPassword:
			hash, err := hex.DecodeString(method.Val1)
			if err != nil {
				return false, AuthDetails{}, err
			}
			didPass = bcrypt.CompareHashAndPassword(hash, []byte(request.FormValue("password")+"yoloSalty"+strconv.Itoa(usr.UID))) == nil
			methodString = "password"
		case user.KindOTP:
			didPass = totp.Validate(request.FormValue("otp"), method.Val1)
			methodString = "OTP"
		}

		switch method.Class {
		case user.ClassAccepted:
			if didPass {
				didPassOne = true
			}
		case user.ClassRequired:
			if !didPass {
				if method.Kind == user.KindOTP {
					details.OTPWanted = true
				}
				return false, details, nil
			}
			didPassOne = true
		}

		if didPass {
			details.PassedMethod = append(details.PassedMethod, methodString)
			switch method.Kind {
			case user.KindPassword:
				details.PassUsed = true
			case user.KindOTP:
				details.OTPUsed = true
			}
		}
	}
	return didPassOne, details, nil
}

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
