// Package redact hides secret values before anything is shown on screen or exported.
package redact

import (
	"math"
	"net/url"
	"regexp"
	"strings"
)

const Mask = "••••"

var (
	// Variable names that usually hold secrets.
	secretKey = regexp.MustCompile(`(?i)(token|secret|password|passwd|pwd|api[_-]?key|apikey|auth|credential|private|session|cookie|bearer|access[_-]?key)`)
	// Value shapes of well-known credentials.
	knownSecret = regexp.MustCompile(`(sk-ant-[A-Za-z0-9_-]{10,}|sk-[A-Za-z0-9_-]{20,}|ghp_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|gh[ousr]_[A-Za-z0-9]{20,}|xox[abprs]-[A-Za-z0-9-]{10,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_-]{30,}|ptr_[A-Za-z0-9+/=_-]{16,}|eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{5,}|-----BEGIN [A-Z ]*PRIVATE KEY-----)`)
	// A reference to an environment variable is not a secret: ${VAR}, $VAR, %VAR%.
	envRef = regexp.MustCompile(`^(\$\{[A-Za-z_][A-Za-z0-9_]*\}|\$[A-Za-z_][A-Za-z0-9_]*|%[A-Za-z_][A-Za-z0-9_]*%)$`)
)

// IsEnvRef reports whether v only points to an environment variable.
func IsEnvRef(v string) bool { return envRef.MatchString(strings.TrimSpace(v)) }

// LooksSecret reports whether a value looks like a credential by shape or entropy.
func LooksSecret(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || IsEnvRef(v) {
		return false
	}
	if knownSecret.MatchString(v) {
		return true
	}
	return len(v) >= 24 && !strings.ContainsAny(v, " /\\:") && entropy(v) >= 4.0
}

// KeyIsSecret reports whether a variable/header name suggests a secret value.
func KeyIsSecret(k string) bool { return secretKey.MatchString(k) }

// Value masks v when its key suggests a secret or the value itself looks like one.
func Value(key, v string) string {
	if v == "" || IsEnvRef(v) {
		return v
	}
	if KeyIsSecret(key) || LooksSecret(v) {
		return Mask
	}
	return URL(v)
}

// Arg masks a command-line argument: '--token=xyz', 'Bearer xyz' and secret-looking values.
func Arg(a string) string {
	if i := strings.Index(a, "="); i > 0 && strings.HasPrefix(a, "-") {
		return a[:i+1] + Value(a[:i], a[i+1:])
	}
	if strings.HasPrefix(strings.ToLower(a), "bearer ") {
		return "Bearer " + Mask
	}
	if LooksSecret(a) {
		return Mask
	}
	return URL(a)
}

// Args masks a full argument list, also the value after flags such as --api-key.
func Args(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if i > 0 && strings.HasPrefix(args[i-1], "-") && !strings.Contains(args[i-1], "=") && KeyIsSecret(args[i-1]) {
			out[i] = Mask
			continue
		}
		out[i] = Arg(a)
	}
	return out
}

// URL removes user:password and secret-looking query parameters from URLs.
func URL(s string) string {
	if !strings.Contains(s, "://") {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if u.User != nil {
		u.User = url.User(Mask)
	}
	q := u.Query()
	changed := false
	for k, vs := range q {
		for i, v := range vs {
			if KeyIsSecret(k) || LooksSecret(v) {
				vs[i] = Mask
				changed = true
			}
		}
	}
	if changed {
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// Text masks known credential shapes anywhere inside a longer text.
func Text(s string) string { return knownSecret.ReplaceAllString(s, Mask) }

func entropy(s string) float64 {
	freq := map[rune]float64{}
	n := 0.0
	for _, r := range s {
		freq[r]++
		n++
	}
	h := 0.0
	for _, c := range freq {
		p := c / n
		h -= p * math.Log2(p)
	}
	return h
}
