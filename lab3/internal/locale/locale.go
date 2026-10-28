// Package locale selects the application's Russian or English messages.
package locale

const (
	Russian = "ru"
	English = "en"
)

func Text(language, russian, english string) string {
	if language == English {
		return english
	}
	return russian
}
