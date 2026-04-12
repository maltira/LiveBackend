package email

import (
	"auth/config"
	"bytes"
	"html/template"
	"time"

	"github.com/go-gomail/gomail"
)

type RecoveryData struct {
	RecoveryURL string
	ExpiresIn   time.Duration
}

var recoveryTmpl *template.Template

func init() {
	tmpl, err := template.ParseFiles("internal/email/templates/recovery_acc.html")
	if err != nil {
		panic("failed to parse email template: " + err.Error())
	}
	recoveryTmpl = tmpl
}

func SendRecovery(toEmail, recoveryURL string, expiresIn time.Duration) error {
	user := config.Env.EmailUsername
	pass := config.Env.EmailPassword

	data := RecoveryData{
		RecoveryURL: recoveryURL,
		ExpiresIn:   expiresIn,
	}

	var body bytes.Buffer
	if err := recoveryTmpl.Execute(&body, data); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", user)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Восстановление аккаунта")
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer("smtp.mail.yahoo.com", 465, user, pass)

	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
