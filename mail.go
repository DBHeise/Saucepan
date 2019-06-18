package main

import (
	"crypto/tls"

	log "github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

func sendMessage(to string, subject string, body string) {

	m := gomail.NewMessage()
	m.SetHeader("From", config.MailConfig.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(config.MailConfig.Server, config.MailConfig.Port, config.MailConfig.User, config.MailConfig.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		log.WithError(err).Warning("Could not send the mail")
	}
}
