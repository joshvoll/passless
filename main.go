package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/matryer/way"
)

// global properties
var config struct {
	port        int
	appURL      *url.URL
	databaseURL string
	jwtKey      []byte
	smtpAddr    string
	smtpAuth    smtp.Auth
}

// init method
func init() {

	// set environment variables
	os.Setenv("SMTP_USERNAME", "db519cb9467011")
	os.Setenv("SMTP_PASSWORD", "9f888a74a8a2f9")

	// configuration of the local db, port, url, smtp, username, password, jwt key, smtp address
	config.port, _ = strconv.Atoi(env("PORT", "8080"))
	puerto := 3000
	config.appURL, _ = url.Parse(env("APP_URL", fmt.Sprintf("http://localhost:%d/", puerto)))
	config.databaseURL = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/passwordless_demo?sslmode=disable")
	config.jwtKey = []byte(env("JWT_KEY", "super-duper-secret-key"))
	smtpHost := env("SMPT_HOST", "smtp.mailtrap.io")
	config.smtpAddr = net.JoinHostPort(smtpHost, env("SMTP_PORT", "25"))
	smtpUsername, ok := os.LookupEnv("SMTP_USERNAME")
	if !ok {
		log.Fatalln("could not find SMTP_USERNAME on the local environments")
	}
	smtpPassword, ok := os.LookupEnv("SMTP_PASSWORD")
	if !ok {
		log.Fatalln("could not find SMTP_PASSWORD on the local environments")
	}

	config.smtpAuth = smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)
}

// env get key and return the environtment base on environment variables, if not return a fallback
func env(key, fallbackValue string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallbackValue
	}

	return v
}

// global properties
var db *sql.DB

func main() {

	fmt.Println(*config.appURL)
	// get the connection to cacroach db
	var err error
	if db, err = sql.Open("postgres", config.databaseURL); err != nil {
		log.Fatalf("could not open to the database : %v\n ", err)
	}

	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("could not ping the database: %v\n ", err)
	}

	// adding the routers, for the router we're using way http sys
	router := way.NewRouter()
	router.HandleFunc("POST", "/api/users", requireJSON(createUser))
	router.HandleFunc("POST", "/api/passwordless/start", requireJSON(passwordLessStart))
	router.HandleFunc("GET", "/api/passwordless/verify_redirect", passwordlessVerifyRedirect)
	router.HandleFunc("GET", "/api/auth_user", guard(getAuthUser))
	router.Handle("GET", "/...", http.FileServer(SPAFileSystem{http.Dir("static")}))

	// run the server
	port := 3000
	log.Printf("starting the server at: %s 🚀 \n ", config.appURL)
	log.Fatalf("could not start the server: %v\n ", http.ListenAndServe(fmt.Sprintf(":%d", port), router))

}
