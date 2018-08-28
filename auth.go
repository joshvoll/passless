package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
)

// global properties
var (
	magicLinkTmpl = template.Must(template.ParseFiles("templates/magic-link.html"))
	keyAuthUserID = ContextKey{"auth_user_id"}
)

// ContextKey provide the name properties
type ContextKey struct {
	Name string
}

func passwordLessStart(w http.ResponseWriter, r *http.Request) {
	// defining the local properties
	var input struct {
		Email       string `json:"email"`
		RedirectURI string `json:"redirectUri"`
	}

	// parse the request
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// validate the request
	errs := make(map[string]string)
	// Email validation options
	if input.Email == "" {
		errs["email"] = "Email is required"
	} else if !rxEmail.MatchString(input.Email) {
		errs["email"] = "Invalid Email format"
	}

	// Redirect validation options
	if input.RedirectURI == "" {
		errs["redirectUri"] = "Redirect URI is required"
	} else if u, err := url.Parse(input.RedirectURI); err != nil || !u.IsAbs() {
		errs["redirectUri"] = "Invalid Email format"
	}

	if len(errs) != 0 {
		respond(w, errs, http.StatusUnprocessableEntity)
		return
	}

	// insert the thing to the db
	var verificationCode string
	err := db.QueryRowContext(r.Context(), `
		INSERT INTO verification_codes (user_id) VALUES
			((SELECT id FROM users WHERE email = $1))
		RETURNING id
	`, input.Email).Scan(&verificationCode)

	// check for erro in the query process
	if errPq, ok := err.(*pq.Error); ok && errPq.Code.Name() == "not_null_violation" {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	} else if err != nil {
		respondInternalError(w, fmt.Errorf("Could not insert verification code: %v ", err))
		return

	}

	// create the url link variable
	q := make(url.Values)
	q.Set("verification_code", verificationCode)
	q.Set("redirect_uri", input.RedirectURI)
	magicLink := *config.appURL
	magicLink.Path = "/api/passwordless/verify_redirect"
	magicLink.RawQuery = q.Encode()

	// parse the body of the html and get ready to send it
	var body bytes.Buffer
	data := map[string]string{"MagicLink": magicLink.String()}
	if err := magicLinkTmpl.Execute(&body, data); err != nil {
		respondInternalError(w, fmt.Errorf("could not execute magic link page: %v", err))
		return
	}

	// send the email to the client
	if err := sendMail(input.Email, "Magic Link", body.String()); err != nil {
		http.Error(w, "could not mail you the magic link. try again please", http.StatusServiceUnavailable)
		return
	}

	// status code
	w.WriteHeader(http.StatusNoContent)

}

func passwordlessVerifyRedirect(w http.ResponseWriter, r *http.Request) {
	// get url query variables and find the verification_code
	q := r.URL.Query()
	verificationCode := q.Get("verification_code")
	redirectURI := q.Get("redirect_uri")

	fmt.Println("VERIFICATION CODE: ", verificationCode)
	// check the errors
	errs := make(map[string]string)
	if verificationCode == "" {
		errs["verification_code"] = "Verification code is required"
	} else if !rxUUID.MatchString(verificationCode) {
		errs["verification_code"] = "Invalid verification code"
	}

	// validate and  get the callback url
	var callback *url.URL
	var err error

	if redirectURI == "" {
		errs["redirect_uri"] = "Redirect is required"
	} else if callback, err = url.Parse(redirectURI); err != nil || !callback.IsAbs() {
		errs["redirect_uri"] = "Invalid Redirect URI"
	}

	if len(errs) != 0 {
		respond(w, errs, http.StatusUnprocessableEntity)
		return
	}

	// save everyting to the data base
	var userID string
	if err := db.QueryRowContext(r.Context(), `
		DELETE FROM verification_codes
		WHERE id = $1
		    AND created_at >= now() - INTERVAL '15m'
		RETURNING user_id
	`, verificationCode).Scan(&userID); err == sql.ErrNoRows {
		http.Error(w, "Link Expired or already userd", http.StatusBadRequest)
		return
	} else if err != nil {
		respondInternalError(w, fmt.Errorf("Could not delete the verification code: %v", err))
		return
	}

	// adding the json web token authentication
	expiresAt := time.Now().Add(time.Hour * 24 * 68)
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Subject:   userID,
		ExpiresAt: expiresAt.Unix(),
	}).SignedString(config.jwtKey)
	if err != nil {
		respondInternalError(w, fmt.Errorf("Could not create JWT: %v", err))
		return
	}

	expiresAtB, err := expiresAt.MarshalText()
	if err != nil {
		respondInternalError(w, fmt.Errorf("could not marshal expritaion date: %v", err))
		return
	}

	f := make(url.Values)
	f.Set("jwt", string(tokenString))
	f.Set("expires_at", string(expiresAtB))
	callback.Fragment = f.Encode()

	// finally redirect the application
	http.Redirect(w, r, callback.String(), http.StatusFound)

}

// withAuth method provide authentication rout
func withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a := r.Header.Get("Authorization")
		hasToken := strings.HasPrefix(a, "Bearer ")
		if !hasToken {
			next(w, r)
			return
		}

		tokenString := a[7:]

		p := jwt.Parser{ValidMethods: []string{jwt.SigningMethodHS256.Name}}
		token, err := p.ParseWithClaims(
			tokenString,
			&jwt.StandardClaims{},
			func(*jwt.Token) (interface{}, error) { return config.jwtKey, nil },
		)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*jwt.StandardClaims)
		if !ok || !token.Valid {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, keyAuthUserID, claims.Subject)

		next(w, r.WithContext(ctx))
	}
}

func guard(next http.HandlerFunc) http.HandlerFunc {
	return withAuth(func(w http.ResponseWriter, r *http.Request) {
		_, ok := r.Context().Value(keyAuthUserID).(string)
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next(w, r)
	})
}

func getAuthUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authUserID := ctx.Value(keyAuthUserID).(string)

	user, err := fetchUser(ctx, authUserID)
	if err == sql.ErrNoRows {
		respond(w, http.StatusText(http.StatusTeapot), http.StatusTeapot)
		return
	} else if err != nil {
		respondInternalError(w, fmt.Errorf("could not query the user: %v", err))
		return
	}

	respond(w, user, http.StatusOK)

}
