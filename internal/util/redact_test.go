package util

import "testing"

func TestRedactURLCredentials(t *testing.T) {
	got := RedactURLCredentials("https://user:pass@example.com/repo.git")
	want := "https://REDACTED@example.com/repo.git"

	if got != want {
		t.Fatalf("unexpected redacted URL\nexpected: %q\nactual:   %q", want, got)
	}
}

func TestRedactSensitive(t *testing.T) {
	got := RedactSensitive(
		`clone failed for https://user:pass@example.com/repo.git with token secret-token`,
		"https://user:pass@example.com/repo.git",
		"secret-token",
	)
	want := `clone failed for https://REDACTED@example.com/repo.git with token [REDACTED]`

	if got != want {
		t.Fatalf("unexpected redacted message\nexpected: %q\nactual:   %q", want, got)
	}
}
