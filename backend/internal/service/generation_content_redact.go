package service

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
)

const generationRedactionVersion = 4

var generationSensitiveKeys = []string{
	"api_key", "api-key", "apikey",
	"x_api_key", "x-api-key", "xapikey",
	"client_secret", "client-secret", "clientsecret",
	"access_token", "access-token", "accesstoken",
	"refresh_token", "refresh-token", "refreshtoken",
	"id_token", "id-token", "idtoken",
	"session_token", "session-token", "sessiontoken",
	"private_key", "private-key", "privatekey",
	"secret_key", "secret-key", "secretkey",
	"access_key", "access-key", "accesskey",
	"token", "session", "cookie", "set_cookie", "set-cookie", "setcookie",
	"authorization", "bearer", "password", "passwd", "pwd", "secret",
}

var (
	generationEmailPattern              = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	generationPhonePattern              = regexp.MustCompile(`\+\d[\d().\-\t ]{6,16}\d|\b\d{2,4}[.\-\t ]\d{3,4}[.\-\t ]\d{3,4}\b|\b1[3-9]\d{9}\b`)
	generationCNIDPattern               = regexp.MustCompile(`\b\d{17}[0-9Xx]\b`)
	generationCardPattern               = regexp.MustCompile(`\b(?:\d[ -]?){12,18}\d\b`)
	generationOpaqueTokenPattern        = regexp.MustCompile(`[A-Za-z0-9]{20,}`)
	generationUnterminatedSecretPattern = compileGenerationUnterminatedSecretPattern(generationSensitiveKeys)
)

func compileGenerationUnterminatedSecretPattern(keys []string) *regexp.Regexp {
	aliases := make([]string, 0, len(keys))
	for _, key := range keys {
		aliases = append(aliases, regexp.QuoteMeta(key))
	}
	return regexp.MustCompile(`(?i)("(?:` + strings.Join(aliases, "|") + `)"\s*:\s*")([^"]*)$`)
}

func redactGenerationPII(value string) string {
	value = generationEmailPattern.ReplaceAllString(value, "[EMAIL]")
	return generationPhonePattern.ReplaceAllString(value, "[PHONE]")
}

func redactGenerationStructuredPII(value string) string {
	value = generationCNIDPattern.ReplaceAllStringFunc(value, func(candidate string) string {
		if isValidGenerationCNID(candidate) {
			return "[ID]"
		}
		return candidate
	})
	value = generationCardPattern.ReplaceAllStringFunc(value, func(candidate string) string {
		if isValidGenerationCard(stripGenerationCardSeparators(candidate)) {
			return "[CARD]"
		}
		return candidate
	})
	return generationOpaqueTokenPattern.ReplaceAllStringFunc(value, func(candidate string) string {
		if looksLikeGenerationOpaqueSecret(candidate) {
			return "[已脱敏]"
		}
		return candidate
	})
}

func stripGenerationCardSeparators(value string) string {
	var result strings.Builder
	result.Grow(len(value))
	for i := 0; i < len(value); i++ {
		if value[i] >= '0' && value[i] <= '9' {
			result.WriteByte(value[i])
		}
	}
	return result.String()
}

func isValidGenerationCard(value string) bool {
	if len(value) < 13 || len(value) > 19 {
		return false
	}
	sum := 0
	double := false
	for i := len(value) - 1; i >= 0; i-- {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
		digit := int(value[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

func isValidGenerationCNID(value string) bool {
	if len(value) != 18 {
		return false
	}
	for i := 0; i < 17; i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	year, _ := strconv.Atoi(value[6:10])
	month, _ := strconv.Atoi(value[10:12])
	day, _ := strconv.Atoi(value[12:14])
	if year < 1900 || year > time.Now().Year() || month < 1 || month > 12 || day < 1 || day > 31 {
		return false
	}
	weights := [...]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checks := [...]byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	sum := 0
	for i := 0; i < 17; i++ {
		sum += int(value[i]-'0') * weights[i]
	}
	got := value[17]
	if got == 'x' {
		got = 'X'
	}
	return got == checks[sum%11]
}

func looksLikeGenerationOpaqueSecret(value string) bool {
	var hasLetter, hasDigit bool
	for i := 0; i < len(value); i++ {
		switch {
		case value[i] >= '0' && value[i] <= '9':
			hasDigit = true
		case value[i] >= 'a' && value[i] <= 'z', value[i] >= 'A' && value[i] <= 'Z':
			hasLetter = true
		}
	}
	return hasLetter && hasDigit
}

func redactGenerationPrompt(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var out string
	if json.Valid(body) {
		out = logredact.RedactJSON(body, generationSensitiveKeys...)
	} else {
		out = logredact.RedactText(string(body), generationSensitiveKeys...)
		out = generationUnterminatedSecretPattern.ReplaceAllString(out, `${1}***`)
	}
	return redactContentModerationSecrets(redactGenerationPII(redactGenerationStructuredPII(out)))
}

func redactGenerationResponse(sample []byte) string {
	if len(sample) == 0 {
		return ""
	}
	return redactContentModerationSecrets(redactGenerationPII(redactGenerationStructuredPII(logredact.RedactText(string(sample)))))
}
