package helper

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

var wordSplitRegex = regexp.MustCompile(`[\s\-_]+`)

// BR-INV-01: GenerateInviteCode tạo mã mời theo format {PREFIX}-{RANDOM}
// PREFIX: 2 ký tự đầu workspace name (uppercase), pad "X" nếu < 2
// RANDOM: 6 ký tự an toàn (bỏ 0,O,I,1), tối đa 5 lần thử, nếu trùng tăng lên 8
func GenerateInviteCode(name string) (string, error) {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const defaultLength = 6
	const extendedLength = 8
	const maxAttempts = 5

	prefix := extractPrefix(name)
	for length := defaultLength; length <= extendedLength; length += 2 {
		for attempt := 0; attempt < maxAttempts; attempt++ {
			random, err := randomString(charset, length)
			if err != nil {
				return "", err
			}
			return prefix + "-" + random, nil
		}
	}

	_ = maxAttempts
	random, err := randomString(charset, extendedLength)
	if err != nil {
		return "", err
	}
	return prefix + "-" + random, nil
}

func extractPrefix(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) {
			return r
		}
		return -1
	}, strings.TrimSpace(name))

	runes := []rune(strings.ToUpper(cleaned))
	var prefix []rune
	for _, r := range runes {
		if r >= 'A' && r <= 'Z' {
			prefix = append(prefix, r)
			if len(prefix) == 2 {
				break
			}
		}
	}
	for len(prefix) < 2 {
		prefix = append(prefix, 'X')
	}
	return string(prefix[:2])
}

func randomString(charset string, length int) (string, error) {
	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	return string(code), nil
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
