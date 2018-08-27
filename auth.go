package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/lib/pq"
)

// global properties
var (
	magicLinkTmpl = template.Must(template.ParseFiles("templates/magic-link.html"))
)

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
	magicLink.Path = "/api/passwordLess/verify_redirect"
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
		errs["redirect_uri"] = "Invalid redirect URI"
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
		    AND create_at >= now() - INTERVAL '15,'
		RETURNING user_id
	`, verificationCode).Scan(&userID); err == sql.ErrNoRows {
		http.Error(w, "Link Expired or already userd", http.StatusBadRequest)
		return
	} else if err != nil {
		respondInternalError(w, fmt.Errorf("Could not delete the verification code: %v", err))
		return
	}

}
