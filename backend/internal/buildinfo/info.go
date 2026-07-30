// Package buildinfo is the single source of deploy identity for Sub2 console.
// HTML injection, public settings, admin GET /version, and CLI --version must
// all derive from the same Info value constructed at process start.
package buildinfo

import (
	"fmt"
	"strings"
	"time"
)

// Info is the immutable build identity for a running process.
type Info struct {
	// Version is the human-facing deploy label: 广州内部版 YYYY.MM.DD-rN
	Version string `json:"version"`
	// BuildCommit is the full git SHA (never force-prefixed with "v").
	BuildCommit string `json:"build_commit"`
	// BuildDate is the build timestamp string from ldflags (may be RFC3339 or date-only).
	BuildDate string `json:"build_date"`
	// BuildType is retained for diagnostics ("source" | "release"); not shown as update affordance.
	BuildType string `json:"build_type,omitempty"`
}

// New constructs Info from ldflag / embedded VERSION inputs and normalizes the display label.
func New(rawVersion, commit, date, buildType string) Info {
	commit = NormalizeCommit(commit)
	date = strings.TrimSpace(date)
	if date == "" {
		date = "unknown"
	}
	label := FormatDisplayLabel(rawVersion, date)
	return Info{
		Version:     label,
		BuildCommit: commit,
		BuildDate:   date,
		BuildType:   strings.TrimSpace(buildType),
	}
}

// NormalizeCommit strips an accidental leading "v" from a SHA-like commit and never adds one.
func NormalizeCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if commit == "" {
		return "unknown"
	}
	// Only strip a single leading "v" when the remainder looks like a hex SHA.
	if len(commit) >= 8 && (commit[0] == 'v' || commit[0] == 'V') {
		rest := commit[1:]
		if isHex(rest) {
			return rest
		}
	}
	return commit
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

// FormatDisplayLabel returns 广州内部版 YYYY.MM.DD-rN from build metadata.
// If rawVersion is already a 广州内部版 label, it is returned unchanged.
func FormatDisplayLabel(rawVersion, buildDate string) string {
	rawVersion = strings.TrimSpace(rawVersion)
	if strings.HasPrefix(rawVersion, "广州内部版 ") {
		return rawVersion
	}
	datePart := FormatDatePart(buildDate)
	rev := ExtractRevision(rawVersion)
	return fmt.Sprintf("广州内部版 %s-r%s", datePart, rev)
}

// FormatDatePart converts a build date string into YYYY.MM.DD.
func FormatDatePart(buildDate string) string {
	buildDate = strings.TrimSpace(buildDate)
	if buildDate == "" || buildDate == "unknown" {
		return "0000.00.00"
	}
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006.01.02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, buildDate); err == nil {
			return t.Format("2006.01.02")
		}
	}
	// Already dotted date
	if len(buildDate) >= 10 && buildDate[4] == '.' && buildDate[7] == '.' {
		return buildDate[:10]
	}
	return "0000.00.00"
}

// ExtractRevision derives rN from a semver-like raw version (patch segment) or falls back to "0".
func ExtractRevision(rawVersion string) string {
	v := strings.TrimSpace(rawVersion)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if v == "" || v == "unknown" {
		return "0"
	}
	parts := strings.Split(v, "-")
	core := parts[0]
	segs := strings.Split(core, ".")
	if len(segs) >= 3 {
		patch := segs[len(segs)-1]
		if patch != "" && isDigits(patch) {
			return patch
		}
	}
	if isDigits(core) {
		return core
	}
	return "0"
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// CLIString formats identity for `server --version`.
func (i Info) CLIString() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", i.Version, i.BuildCommit, i.BuildDate)
}

// PublicMap is the JSON object shared by public settings and admin version responses.
func (i Info) PublicMap() map[string]string {
	return map[string]string{
		"version":      i.Version,
		"build_commit": i.BuildCommit,
		"build_date":   i.BuildDate,
	}
}
