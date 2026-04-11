package email

import (
	"auth/config"
	"bytes"
	"html/template"
	"time"

	"github.com/go-gomail/gomail"
)

type PassResetData struct {
	ResetURL  string
	ExpiresIn time.Duration
}

var passResetTmpl *template.Template

func init() {
	tmpl, err := template.ParseFiles("internal/email/templates/reset_pass.html")
	if err != nil {
		panic("failed to parse email template: " + err.Error())
	}
	passResetTmpl = tmpl
}

func SendPasswordReset(toEmail, resetURL string, expiresIn time.Duration) error {
	user := config.Env.EmailUsername
	pass := config.Env.EmailPassword

	data := PassResetData{
		ResetURL:  resetURL,
		ExpiresIn: expiresIn,
	}

	var body bytes.Buffer
	if err := passResetTmpl.Execute(&body, data); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", user)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Сброс пароля — Live Messenger")
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer("smtp.mail.yahoo.com", 465, user, pass)

	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
