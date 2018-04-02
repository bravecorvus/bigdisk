package email

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

type Mail struct {
	senderId string
	toIds    []string
	subject  string
	body     string
}

type SmtpServer struct {
	host string
	port string
}

func (s *SmtpServer) ServerName() string {
	return s.host + ":" + s.port
}

func (mail *Mail) BuildMessage() string {
	message := ""
	message += fmt.Sprintf("From: %s\r\n", mail.senderId)
	if len(mail.toIds) > 0 {
		message += fmt.Sprintf("To: %s\r\n", strings.Join(mail.toIds, ";"))
	}

	message += fmt.Sprintf("Subject: %s\r\n", mail.subject)
	message += "\r\n" + mail.body

	return message
}

func SendEmail(to, subject, message string) (bool, string) {
	from := os.Getenv("BIGDISKEMAILUSERNAME")
	pass := os.Getenv("BIGDISKEMAILPASSWORD")
	if from == "" || pass == "" {
		return false, "BIGDISKEMAILUSERNAME and/or BIGDISKEMAILPASSWORD not defined in environment variables"
	}
	toarr := []string{}
	toarr = append(toarr, to)
	mail := Mail{from, toarr, subject, message}
	messageBody := mail.BuildMessage()
	smtpServer := SmtpServer{host: "smtp.gmail.com", port: "465"}
	auth := smtp.PlainAuth("", mail.senderId, pass, smtpServer.host)

	// Gmail will reject connection if it's not secure
	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         smtpServer.host,
	}

	conn, err := tls.Dial("tcp", smtpServer.ServerName(), tlsconfig)
	if err != nil {
		log.Println(err.Error())
		return false, "conn, err := tls.Dial(\"tcp\", smtpServer.ServerName(), tlsconfig)"
	}

	client, err := smtp.NewClient(conn, smtpServer.host)
	if err != nil {
		log.Println(err.Error())
		return false, "client, err := smtp.NewClient(conn, smtpServer.host)"
	}

	// step 1: Use Auth
	if err = client.Auth(auth); err != nil {
		log.Println(err.Error())
		return false, "err = client.Auth(auth); err != nil {"
	}

	// step 2: add all from and to
	if err = client.Mail(mail.senderId); err != nil {
		log.Println(err.Error())
		return false, "if err = client.Mail(mail.senderId); err != nil {"
	}
	for _, k := range mail.toIds {
		if err = client.Rcpt(k); err != nil {
			log.Println(err.Error())
			return false, "if err = client.Rcpt(k); err != nil {"
		}
	}

	// Data
	w, err := client.Data()
	if err != nil {
		log.Println(err.Error())
		return false, "w, err := client.Data()"
	}

	_, err = w.Write([]byte(messageBody))
	if err != nil {
		log.Println(err.Error())
		return false, "_, err = w.Write([]byte(messageBody))"
	}

	err = w.Close()
	if err != nil {
		log.Println(err.Error())
		return false, "err = w.Close()"
	}

	err = client.Quit()
	if err != nil {
		log.Println(err.Error())
		return false, "err = client.Quit()"
	}

	return true, "sucess"
}
