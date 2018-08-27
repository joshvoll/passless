package main

import (
	"fmt"
	"net/mail"
	"net/smtp"
)

func sendMail(to string, subject string, body string) error {
	toAddr := mail.Address{Address: to}
	fromAddr := mail.Address{
		Name:    "Passwordless Demo",
		Address: "noreply@" + config.appURL.Host,
	}

	headers := map[string]string{
		"From":         fromAddr.String(),
		"To":           toAddr.String(),
		"Subject":      subject,
		"Content-Type": `text/html; charset=utf-8`,
	}

	msg := ""
	for k, v := range headers {
		msg += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	msg += "\r\n"
	msg += body

	// send the email
	return smtp.SendMail(
		config.smtpAddr,
		config.smtpAuth,
		fromAddr.Address,
		[]string{toAddr.Address},
		[]byte(msg))
}
