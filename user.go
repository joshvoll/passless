package main

import (
	"encoding/json"
	"net/http"
	"regexp"
)

// User struct
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

// global variables
var (
	rxEmail    = regexp.MustCompile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")
	rxUserName = regexp.MustCompile("^[a-zA-Z][\\w|-]{0,17}$")
)

func createUser(w http.ResponseWriter, r *http.Request) {
	// local properties
	var user User

	// get the response body and add it to the struct
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		respondJSON(w, err.Error(), http.StatusBadRequest)
		return
	}

	// defer the body
	defer r.Body.Close()

	// check if there is an error on the struct
	errs := make(map[string]string)

	// check if the email have error
	if user.Email == "" {
		errs["email"] = "Email is required"

	} else if !rxEmail.MatchString(user.Email) {
		errs["email"] = "Invalid email"

	}

	// check the user name error struct
	if user.UserName == "" {
		errs["username"] = "User name is required"

	} else if !rxUserName.MatchString(user.UserName) {
		errs["username"] = "User name is invalid"

	}

	// send all the errors to the response payload
	if len(errs) != 0 {
		respondJSON(w, errs, http.StatusUnprocessableEntity)
		return
	}

	// add the user to the database

}
