package email

import (
	"auth/config"
	"bytes"
	"html/template"

	"github.com/go-gomail/gomail"
)

type OtpData struct {
	Code      string
	ExpiresIn string
}

var otpTmpl *template.Template

func init() {
	tmpl, err := template.ParseFiles("internal/email/templates/otp.html")
	if err != nil {
		panic("failed to parse email template: " + err.Error())
	}
	otpTmpl = tmpl
}

func SendOTP(toEmail, code string, exp string) error {
	user := config.Env.EmailUsername
	pass := config.Env.EmailPassword

	data := OtpData{
		Code:      code,
		ExpiresIn: exp,
	}

	var body bytes.Buffer
	if err := otpTmpl.Execute(&body, data); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", user)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Ваш код подтверждения — "+code)
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer("smtp.mail.yahoo.com", 465, user, pass)

	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
