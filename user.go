package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/lib/pq"
)

// global properties
var (
	rxEmail    = regexp.MustCompile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")
	rxUserName = regexp.MustCompile("^[a-zA-Z][\\w|-]{0,17}$")
	rxUUID     = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")
)

// User data definition
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

func createUser(w http.ResponseWriter, r *http.Request) {
	// assign the user struct
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()
	// get the payload and add it to the struct
	errs := make(map[string]string)
	if user.Email == "" {
		errs["email"] = "Email is required"
	} else if !rxEmail.MatchString(user.Email) {
		errs["email"] = "Invalid Email format"
	}

	if user.UserName == "" {
		errs["username"] = "User Name is required"
	} else if !rxUserName.MatchString(user.UserName) {
		errs["username"] = "Invalid User Name"
	}

	if len(errs) != 0 {
		respond(w, errs, http.StatusUnprocessableEntity)
		return
	}

	// save everything to the database
	err := db.QueryRowContext(r.Context(), `
		INSERT INTO users (email, username) VALUES ($1, $2)
		RETURNING id
	`, user.Email, user.UserName).Scan(&user.ID)

	// validate the db errors
	if errPq, ok := err.(*pq.Error); ok && errPq.Code.Name() == "unique_violation" {
		if strings.Contains(errPq.Error(), "email") {
			errs["email"] = "Email is Taken"
		} else {
			errs["username"] = "User Name is taken"
		}

		respond(w, errs, http.StatusForbidden)
		return

	} else if err != nil {
		respondInternalError(w, fmt.Errorf("could not insert user %v ", err))
		return
	}

	// response the payload
	respond(w, user, http.StatusCreated)
}

func fetchUser(ctx context.Context, id string) (User, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	user := User{ID: id}
	err := db.QueryRowContext(ctx, `
		SELET emai, username FROM users WHERE id = $1
	`, id).Scan(&user.Email, &user.UserName)

	user.ID = id
	return user, err
}
