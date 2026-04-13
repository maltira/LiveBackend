package custom

import (
	"errors"
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("strong_password", validateStrongPassword)
		if err != nil {
			panic(err)
		}
	}
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>/?]`).MatchString(password)

	return hasUpper && hasLower && hasDigit && hasSpecial
}

func IsValidationError(err error) (string, bool) {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return "", false
	}
	fe := ve[0]
	switch fe.Tag() {
	case "required":
		return "Заполните все обязательные поля", true
	case "email":
		return "Некорректный формат email адреса", true
	case "len":
		return "OTP-код имеет неверный формат", true
	case "strong_password":
		return "Пароль должен содержать мин. 8 символов: заглавные и строчные буквы, цифры и специальный символ", true
	case "nefield":
		return "Новый пароль не должен совпадать со старым", true
	case "eqfield":
		return "Пароли должны совпадать", true
	default:
		return fe.Error(), true // fallback
	}
}
