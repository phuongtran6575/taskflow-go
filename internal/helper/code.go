package helper

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

var wordSplitRegex = regexp.MustCompile(`[\s\-_]+`)

func GenerateInviteCode() (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const prefix = "WS-"
	const length = 6

	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	return prefix + string(code), nil
}

func GenerateProjectKey(name string) string {
	words := wordSplitRegex.Split(strings.TrimSpace(name), -1)
	var filtered []string
	for _, w := range words {
		if w != "" {
			filtered = append(filtered, w)
		}
	}

	var key string
	if len(filtered) >= 2 {
		maxWords := 5
		if len(filtered) < maxWords {
			maxWords = len(filtered)
		}
		for i := 0; i < maxWords; i++ {
			runes := []rune(strings.TrimSpace(filtered[i]))
			if len(runes) > 0 {
				key += string(unicode.ToUpper(runes[0]))
			}
		}
	} else if len(filtered) == 1 {
		clean := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, filtered[0])
		if len(clean) > 5 {
			clean = clean[:5]
		}
		key = strings.ToUpper(clean)
	}

	key = strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, strings.ToUpper(key))

	if len(key) < 2 {
		key = "PRJ"
	}
	if len(key) > 10 {
		key = key[:10]
	}
	return key
}
