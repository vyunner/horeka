package auth

import (
	"strings"
	"unicode"
)

func validatePhone(s string) (string, bool) {
	s = strings.TrimSpace(s)

	if len(s) != 11 {
		return "", false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return "", false
		}
	}
	if s[0] != '7' {
		return "", false
	}
	return s, true
}
