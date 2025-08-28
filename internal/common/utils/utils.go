package utils

import (
	"strings"
	"unicode"
)

// CleanString удаляет все не-буквенные и не-цифровые символы
func CleanString(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return strings.ToLower(result.String())
}
