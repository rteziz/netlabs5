package locale

const (
	Russian = "ru"
	English = "en"
)

// Text выбирает текст сообщения по языку приложения.
func Text(language, russian, english string) string {
	if language == English {
		return english
	}
	return russian
}
