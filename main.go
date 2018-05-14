package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	"github.com/matryer/way"
)

// config variables struct
var config struct {
	port        int
	appURL      *url.URL
	databaseURL string
	jwtKey      []byte
	smtpAdress  string
	smtpAuth    smtp.Auth
}

// init method
func init() {

	// set local environments
	os.Setenv("SMTP_USERNAME", "db519cb9467011")
	os.Setenv("SMTP_PASSWORD", "9f888a74a8a2f9")

	// configuration ports
	config.port, _ = strconv.Atoi(env("PORT", "8080"))
	config.appURL, _ = url.Parse(env("APP_URL", "http://localhost:"+strconv.Itoa(config.port)+"/"))
	config.databaseURL = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/passwordless_demo?sslmode=disable")
	config.jwtKey = []byte(env("JWT_KEY", "super-duper-secret-key"))
	smtpHost := env("SMTP_HOST", "smtp.mailtrap.io")
	config.smtpAdress = net.JoinHostPort(smtpHost, env("SMTP_HOST", "25"))
	smtpUsername, ok := os.LookupEnv("SMTP_USERNAME")
	if !ok {
		log.Fatalln("could not find SMTP_USERNAME on environment variables")
	}
	smtpPassword, ok := os.LookupEnv("SMTP_PASSWORD")
	if !ok {
		log.Fatalln("could not find SMTP_PASSWORD on environment variables")
	}

	// config the smtp server
	config.smtpAuth = smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)

}

var db *sql.DB

func env(key, fallbackValue string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallbackValue
	}

	return v
}

func main() {
	// defining local properties
	var err error

	// db connection
	if db, err = sql.Open("postgres", config.databaseURL); err != nil {
		log.Fatalf("Could not open the db connection: %v\n", err)
	}

	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Could not pin the DB: %v\n", err)
	}
	// creating the router
	router := way.NewRouter()
	router.HandleFunc("POST", "/api/passless/start", jsonRequired(passlessStart))
	router.HandleFunc("GET", "/api/passless/verify_redirect", passlessVerifyRedirect)

	addr := fmt.Sprintf(":%d", config.port)
	log.Printf("Starting server at %s 🚀 \n", config.appURL)
	log.Fatalf("could not start the server: %v \n", http.ListenAndServe(addr, router))

}

func passlessStart(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func passlessVerifyRedirect(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
}

func jsonRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		isJSON := strings.HasPrefix(ct, "application/json")
		if !isJSON {
			respondJSON(w, "JSON Body is required", http.StatusUnsupportedMediaType)
			return

		}
		next(w, r)
	}
}

func respondJSON(w http.ResponseWriter, payload interface{}, code int) {
	switch value := payload.(type) {
	case string:
		payload = map[string]string{"message": value}
	case int:
		payload = map[string]int{"value": value}

	case bool:
		payload = map[string]bool{"result": value}
	}

	// convert everrtying to json marshal
	b, err := json.Marshal(payload)
	if err != nil {
		respondInternalError(w, fmt.Errorf("could not marshal the response code payload: %v: ", err))
		return
	}

	// put header on the response code
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(b)
}

func respondInternalError(w http.ResponseWriter, err error) {
	log.Println(err)
	respondJSON(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	return
}
