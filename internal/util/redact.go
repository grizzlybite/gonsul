package util

import (
	"net/url"
	"strings"
)

const redacted = "[REDACTED]"
const redactedURLUser = "REDACTED"

func RedactURLCredentials(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.User == nil {
		return rawURL
	}

	parsedURL.User = url.User(redactedURLUser)
	return parsedURL.String()
}

func RedactSensitive(message string, sensitiveValues ...string) string {
	for _, value := range sensitiveValues {
		if value == "" {
			continue
		}

		replacement := redacted
		if redactedURL := RedactURLCredentials(value); redactedURL != value {
			replacement = redactedURL
		}

		message = strings.ReplaceAll(message, value, replacement)
	}

	return message
}
