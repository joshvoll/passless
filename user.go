package main

import "regexp"

// User struct for the user authentication
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

// global properties
var (
	rxMail     = regexp.MustCompile("^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$")
	rxUserName = regexp.MustCompile("^[a-zA-Z][\\w|-]{0,17}$")
)
