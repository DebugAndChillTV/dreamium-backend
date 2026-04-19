package routes

import (
	"errors"
	"strings"

	"github.com/getsentry/sentry-go"
)

const maxDreamLength = 2000

// suspiciousPatterns are logged to Sentry but do NOT cause rejection.
var suspiciousPatterns = []string{
	"ignore previous",
	"ignore all previous",
	"you are now",
	"forget your instructions",
	"new system prompt",
	"system prompt",
	"system:",
	"disregard",
	"jailbreak",
	"act as if",
	"pretend you are",
	"override",
	"no restrictions",
	"unrestricted",
}

// sanitizeDreamInput validates user dream text.
// Returns the input unchanged on success, the list of suspicious patterns matched (non-empty means log to Sentry),
// and a non-nil error only when the input must be rejected hard.
func sanitizeDreamInput(input string) (string, []string, error) {
	if len([]rune(input)) > maxDreamLength {
		return "", nil, errors.New("dream input exceeds 2000 character limit")
	}

	lower := strings.ToLower(input)
	if strings.Contains(lower, "</user_dream>") {
		return "", nil, errors.New("invalid dream input")
	}

	var matched []string
	for _, p := range suspiciousPatterns {
		if strings.Contains(lower, p) {
			matched = append(matched, p)
		}
	}

	return input, matched, nil
}

// reportSuspiciousInput sends a Sentry event with pattern metadata only — never dream content.
func reportSuspiciousInput(userID, endpoint string, patterns []string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("endpoint", endpoint)
		scope.SetTag("user_id", userID)
		scope.SetExtra("matched_patterns", patterns)
		sentry.CaptureMessage("suspicious dream input detected")
	})
}
