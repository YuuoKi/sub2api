package service

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// volcengineAccessKeyPattern catches Volcengine-style access keys that begin
// with the AKLT prefix and carry no delimiter — a mid-length, mixed-case,
// non-hex shape that the shared content-moderation patterns miss (they require
// a delimiter, >=48 chars, or pure hex). Kept video-local so the shared
// redactor is untouched. Tighten/confirm against the real key shape at smoke.
var volcengineAccessKeyPattern = regexp.MustCompile(`(?i)\bAKLT[A-Za-z0-9+/=_-]{16,}\b`)

// videoOpaqueTokenMinLen is the floor for the VIDEO-LOCAL opaque-token rule.
//
// The shared content-moderation catch-alls only fire on a delimiter-less blob at
// >=48 chars (idx6 `[A-Za-z0-9_-]{48,}`, idx7 base64) or on pure hex (idx5), and
// the field-name rule (idx1 `\b(field)\s*[:=]`) misses JSON like {"api_key":"x"}
// because the quote between key and colon breaks its `\b...[:=]` anchor. That
// leaves a real gap: an opaque, mixed-case, non-hex, delimiter-less credential
// of ~20-47 chars with no recognized prefix passes through UNREDACTED into
// task.ErrorMessage (DB), video_task_events.Message (DB), the API error response
// and the redacted audit log. 20 is high enough to stay above ordinary
// identifiers/words and low enough to cover that band. Credentials of 12-19
// chars remain a residual gap by design (too short to redact without chewing
// through legitimate short ids); the real-smoke PRE-ARM self-check is the
// backstop that aborts before any billed call if the configured key shape is not
// actually stripped.
const videoOpaqueTokenMinLen = 20

// videoOpaqueTokenCandidate finds delimiter-less alphanumeric runs long enough to
// plausibly be a credential. It is intentionally alnum-only (no +/=_-.): hyphen/
// underscore-delimited strings are already handled by the shared patterns, and
// restricting to alnum keeps the rule from chewing through hyphenated/dotted
// identifiers and URLs (full http(s) URLs are already redacted by the shared
// idx0 rule before this pass runs). Whether a candidate is an actual secret is
// decided in looksLikeOpaqueVideoSecret, because RE2 has no lookahead to express
// "a 20+ run that contains at least one digit AND one letter".
var videoOpaqueTokenCandidate = regexp.MustCompile(fmt.Sprintf(`[A-Za-z0-9]{%d,}`, videoOpaqueTokenMinLen))

// looksLikeOpaqueVideoSecret reports whether a delimiter-less alphanumeric run is
// shaped like an opaque credential rather than an ordinary identifier or word.
// The discriminator is alphanumeric mixing: a long run carrying BOTH a digit and
// a letter is the hallmark of a random token, whereas natural-language words and
// CamelCase identifiers carry no digits and numeric ids/timestamps/counters carry
// no letters. Pure-digit runs and pure-letter runs are therefore left
// inspectable, which is what keeps result video_url path segments, model names,
// task ids and ordinary error prose readable for debugging.
//
// An all-letter, digit-free credential would slip past this rule; that residual
// risk is covered by the real-smoke harness's fail-closed PRE-ARM self-check,
// which aborts before any billed call if the configured key is not stripped in
// every echo shape.
func looksLikeOpaqueVideoSecret(tok string) bool {
	if len(tok) < videoOpaqueTokenMinLen {
		return false
	}
	var hasDigit, hasLetter bool
	for i := 0; i < len(tok); i++ {
		switch c := tok[i]; {
		case c >= '0' && c <= '9':
			hasDigit = true
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			hasLetter = true
		}
		if hasDigit && hasLetter {
			return true
		}
	}
	return false
}

// redactVideoUpstreamSecrets strips credentials (Bearer tokens, Authorization
// headers, API keys, JWTs, presigned URLs, long hex/base64 blobs, Volcengine
// AKLT access keys) from upstream video-provider response bodies before they
// are persisted to the DB, surfaced in API responses, or written to the
// redacted event log.
//
// It deliberately reuses the shared, battle-tested secret patterns from the
// content-moderation redactor (same package) and layers the Volcengine-
// specific pattern on top. The redaction is intentionally aggressive: for a
// smoke-phase upstream body that is about to be stored and returned to the
// frontend, over-redaction is the safe failure mode — a leaked API key is far
// costlier than a stripped diagnostic URL.
func redactVideoUpstreamSecrets(s string) string {
	out := redactContentModerationSecrets(s)
	out = volcengineAccessKeyPattern.ReplaceAllString(out, "[已脱敏]")
	// VIDEO-LOCAL final pass: strip delimiter-less opaque alnum tokens (>=20
	// chars carrying both a digit and a letter) that escape every shared pattern
	// and the AKLT rule. Decided per-candidate in code because RE2 has no
	// lookahead. Runs last so it only sees what the prior passes left behind.
	out = videoOpaqueTokenCandidate.ReplaceAllStringFunc(out, func(tok string) string {
		if looksLikeOpaqueVideoSecret(tok) {
			return "[已脱敏]"
		}
		return tok
	})
	return out
}

// appendRedactedVideoEvent appends a single redacted JSON line describing a real
// upstream interaction to the audit log at SUB2API_VIDEO_REDACTED_EVENT_LOG.
//
// The smoke gate (seedanceSmokeGateBlockedReasons) requires this env var to be
// non-empty before any real Seedance call is allowed. Before this writer existed
// the env var was a tripwire-only precondition and `redacted_event:true` was a
// bare payload flag — i.e. the "redacted event log" the gate promised did not
// actually exist. This function makes that promise real:
//
//   - the body is redacted via redactVideoUpstreamSecrets before writing;
//   - the file is opened append-only with 0600 (owner read/write only);
//   - if the env var is empty it is a no-op (never reached on the real path,
//     because the gate already blocks real calls when it is empty).
//
// Operators MUST point SUB2API_VIDEO_REDACTED_EVENT_LOG at a git-ignored path
// (a `*.log` file or under backend/data/ — both are covered by .gitignore).
//
// It is FAIL-CLOSED: the smoke gate requires the env var, and the audit trail is
// a security precondition for real calls, so any failure to record it returns an
// error (the caller aborts the real create/poll) rather than silently proceeding
// with no audit evidence.
func appendRedactedVideoEvent(phase string, statusCode int, rawBody string) error {
	path := strings.TrimSpace(os.Getenv("SUB2API_VIDEO_REDACTED_EVENT_LOG"))
	if path == "" {
		return nil
	}

	record := map[string]any{
		"ts":          time.Now().UTC().Format(time.RFC3339Nano),
		"provider":    VideoProviderSeedance,
		"phase":       phase,
		"status_code": statusCode,
		"body":        truncate(redactVideoUpstreamSecrets(rawBody), 1000),
	}
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal redacted event: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open redacted event log: %w", err)
	}
	defer f.Close()
	// os.OpenFile only applies the mode on creation; enforce 0600 explicitly so a
	// pre-existing world/group-readable file (e.g. 0644) cannot leak the audit
	// trail. Fail-closed: if we cannot guarantee 0600, abort rather than write
	// the audit line to a potentially over-permissive file.
	if err := f.Chmod(0o600); err != nil {
		return fmt.Errorf("enforce 0600 on redacted event log: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write redacted event log: %w", err)
	}
	return nil
}
