package service

import "strings"

func sanitizeBatchImageProviderFailure(code, message, fallbackCode, fallbackMessage string) (string, string) {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	if !isSecretSafeProviderErrorCode(code) {
		code = fallbackCode
	}
	if !isSecretSafeProviderErrorMessage(message) {
		message = fallbackMessage
	}
	return code, message
}

func isSecretSafeProviderErrorCode(value string) bool {
	if value == "" || len(value) > 128 || containsSecretLikeProviderText(value) {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '.':
		default:
			return false
		}
	}
	return true
}

func isSecretSafeProviderErrorMessage(value string) bool {
	if value == "" || len(value) > batchImageMaxErrorMessageLength || containsSecretLikeProviderText(value) {
		return false
	}
	lower := strings.ToLower(value)
	return !(strings.Contains(lower, "://") && strings.Contains(lower, "?"))
}

func containsSecretLikeProviderText(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"authorization", "bearer ", "api_key", "apikey", "access_token",
		"refresh_token", "id_token", "token=", "token:", "secret",
		"password", "credential", "x-amz-", "signature=", "signed-url",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
