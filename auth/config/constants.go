package config

const (
	IncorrectDataError = "Переданы некорректные данные в запросе"
	UnauthorizedError  = "Требуется авторизация"
	InvalidOtpError    = "OTP-код истёк или не существует"
	InvalidTokenError  = "Токен не существует или срок действия истёк"
	IncorrectUUIDError = "Некорректный формат UUID"
	IncorrectAuthError = "Неверный email или пароль"
	NotFoundError      = "Запись не найдена"
	NotVerifiedError   = "Подтвердите указанную почту на этапе регистрации"
)
