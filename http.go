package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// requireJSON GET: http => validate json => http respond
func requireJSON(next http.HandlerFunc) http.HandlerFunc {
	// return the handler
	return func(w http.ResponseWriter, r *http.Request) {
		// validate the header to make sure is json format
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "Content-Type application/json is REQUIRED ", http.StatusUnsupportedMediaType)
			return
		}
		// send the request if everything is good
		next(w, r)
	}
}

// respond http responde, payload, status code => erro || respond json format
func respond(w http.ResponseWriter, payload interface{}, statusCode int) {
	b, err := json.Marshal(payload)
	if err != nil {
		respondInternalError(w, fmt.Errorf("could not marshal de payload: %v ", err))
		return
	}

	// add the json format to the headsr
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	w.Write(b)
}

// respondInternalError  http responde, err => error
func respondInternalError(w http.ResponseWriter, err error) {
	// log the error for splunk
	fmt.Println(err)

	// response the error to eh payload
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	return
}
