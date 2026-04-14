package email

import (
	"auth/config"
	"bytes"
	"html/template"

	"github.com/go-gomail/gomail"
)

type VerificationData struct {
	VerifyURL string
	ExpiresIn string // "15 минут"
}

var verificationTmpl *template.Template

func init() {
	tmpl, err := template.ParseFiles("internal/email/templates/verification.html")
	if err != nil {
		panic("failed to parse email template: " + err.Error())
	}
	verificationTmpl = tmpl
}

func SendVerificationEmail(toEmail, verifyURL string) error {
	user := config.Env.EmailUsername
	pass := config.Env.EmailPassword

	data := VerificationData{
		VerifyURL: verifyURL,
		ExpiresIn: "15 минут",
	}

	var body bytes.Buffer
	if err := verificationTmpl.Execute(&body, data); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", user)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Подтвердите ваш аккаунт на Live Messenger")
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer("smtp.mail.yahoo.com", 465, user, pass)

	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
