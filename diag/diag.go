// Package diag describes problems found in user-provided data: tapir.json,
// API specifications, env files.
//
// A Diagnostic is not a Go error. An error means an operation could not be
// performed at all (a file could not be read); diagnostics mean the operation
// was performed, but the data has problems. Functions that inspect user data
// return every problem they find as a List instead of stopping at the first
// one, so users can fix them all at once.
//
// Diagnostics are static: they are produced before a run. Failures that
// happen during a run (failed assertions, timeouts) belong to the report.
package diag

import (
	"fmt"
	"strings"
)

// Severity tells how serious a problem is.
type Severity int

const (
	// Error blocks the run: the data is invalid.
	Error Severity = iota + 1
	// Warning does not block the run, but likely indicates a mistake.
	Warning
	// Info is purely informational.
	Info
)

var severityNames = map[Severity]string{
	Error:   "error",
	Warning: "warning",
	Info:    "info",
}

//goland:noinspection GoMixedReceiverTypes
func (s Severity) String() string {
	if name, ok := severityNames[s]; ok {
		return name
	}
	return fmt.Sprintf("Severity(%d)", int(s))
}

// MarshalText encodes the severity as its lowercase name.
//
//goland:noinspection GoMixedReceiverTypes
func (s Severity) MarshalText() ([]byte, error) {
	name, ok := severityNames[s]
	if !ok {
		return nil, fmt.Errorf("diag: unknown severity %d", int(s))
	}
	return []byte(name), nil
}

// UnmarshalText decodes a severity from its lowercase name.
//
//goland:noinspection GoMixedReceiverTypes
func (s *Severity) UnmarshalText(text []byte) error {
	for sev, name := range severityNames {
		if name == string(text) {
			*s = sev
			return nil
		}
	}
	return fmt.Errorf("diag: unknown severity %q", text)
}

// Code is a stable identifier of a kind of problem, e.g. "TAPIR-E012".
//
// Codes are declared as constants by the package that reports them. They are
// used by UIs for translation, by documentation and by tests, so a code must
// never change its meaning once published.
type Code string

// Diagnostic is a single problem found in user data.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     Code     `json:"code"`
	Location Location `json:"location"`
	// Message is a human-readable English description of the problem.
	Message string `json:"message"`
}

// Errorf creates a diagnostic with Error severity.
func Errorf(code Code, loc Location, format string, args ...any) Diagnostic {
	return newf(Error, code, loc, format, args...)
}

// Warningf creates a diagnostic with Warning severity.
func Warningf(code Code, loc Location, format string, args ...any) Diagnostic {
	return newf(Warning, code, loc, format, args...)
}

// Infof creates a diagnostic with Info severity.
func Infof(code Code, loc Location, format string, args ...any) Diagnostic {
	return newf(Info, code, loc, format, args...)
}

func newf(sev Severity, code Code, loc Location, format string, args ...any) Diagnostic {
	return Diagnostic{
		Severity: sev,
		Code:     code,
		Location: loc,
		Message:  fmt.Sprintf(format, args...),
	}
}

// String formats the diagnostic for humans:
//
//	tapir.json:148:17: error TAPIR-E012: unknown input "age"
func (d Diagnostic) String() string {
	var b strings.Builder
	if loc := d.Location.String(); loc != "" {
		b.WriteString(loc)
		b.WriteString(": ")
	}
	b.WriteString(d.Severity.String())
	if d.Code != "" {
		b.WriteString(" ")
		b.WriteString(string(d.Code))
	}
	b.WriteString(": ")
	b.WriteString(d.Message)
	return b.String()
}
