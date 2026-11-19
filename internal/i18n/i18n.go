// Package i18n translates catude's interface.
//
// Messages are looked up by their English text (gettext style): T("Sessions") returns the translation in
// the current language, or the English text when there is none, so an untranslated message is never
// blank. F works like fmt.Sprintf with a translated format. The catalogs live in locales/<code>.json;
// scripts/i18n-extract.mjs lists every message used in the code (locales/messages.json) and the tests
// check that every catalog covers it with the same placeholders.
//
// The translations follow the terminology of established localized software: Anthropic's own localized
// Claude Code documentation for Claude terms (skills, hooks, subagents, context window…), Visual Studio
// Code's language packs (Microsoft terminology) for general interface words, and lazygit's translations
// for terminal-UI conventions. See docs/translations.md.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed locales/*.json
var files embed.FS

// Language is one supported interface language.
type Language struct {
	Code   string // BCP 47
	Name   string // in its own language
	CJK    bool   // needs a font with Chinese, Japanese or Korean characters
	Native string // English name, for documentation
}

// Languages are the supported languages, English first.
var Languages = []Language{
	{"en", "English", false, "English"},
	{"es", "Español", false, "Spanish"},
	{"ja", "日本語", true, "Japanese"},
	{"zh-Hans", "简体中文", true, "Chinese (Simplified)"},
	{"zh-Hant", "繁體中文", true, "Chinese (Traditional)"},
	{"ko", "한국어", true, "Korean"},
	{"de", "Deutsch", false, "German"},
	{"fr", "Français", false, "French"},
	{"pt-BR", "Português (Brasil)", false, "Portuguese (Brazil)"},
	{"ru", "Русский", false, "Russian"},
}

var (
	mu   sync.RWMutex
	cur  = "en"
	msgs map[string]string
)

// Set chooses the interface language ("auto" or "" detects it). Unknown codes fall back to English.
func Set(code string) {
	if code == "" || code == "auto" {
		code = Detect()
	}
	code = Match(code)
	m := map[string]string{}
	if code != "en" {
		if b, err := files.ReadFile("locales/" + code + ".json"); err == nil {
			_ = json.Unmarshal(b, &m)
		}
	}
	mu.Lock()
	cur, msgs = code, m
	mu.Unlock()
}

// Lang is the current language code.
func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return cur
}

// Current is the current language.
func Current() Language {
	l := Lang()
	for _, x := range Languages {
		if x.Code == l {
			return x
		}
	}
	return Languages[0]
}

// T translates a message.
func T(s string) string {
	mu.RLock()
	defer mu.RUnlock()
	if t, ok := msgs[s]; ok && t != "" {
		return t
	}
	return s
}

// F translates a format and fills it like fmt.Sprintf.
func F(format string, args ...any) string { return fmt.Sprintf(T(format), args...) }

// Match maps a locale name (ja_JP.UTF-8, zh-TW, pt_BR, es-419, chinese…) to a supported code.
func Match(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexAny(s, ".@"); i >= 0 {
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "_", "-")
	switch {
	case s == "":
		return "en"
	case strings.HasPrefix(s, "zh"):
		if strings.Contains(s, "hant") || strings.HasSuffix(s, "-tw") || strings.HasSuffix(s, "-hk") || strings.HasSuffix(s, "-mo") {
			return "zh-Hant"
		}
		return "zh-Hans"
	case strings.HasPrefix(s, "pt"):
		return "pt-BR"
	}
	names := map[string]string{"english": "en", "spanish": "es", "español": "es", "espanol": "es", "japanese": "ja", "日本語": "ja",
		"chinese": "zh-Hans", "中文": "zh-Hans", "korean": "ko", "한국어": "ko", "german": "de", "deutsch": "de", "french": "fr",
		"français": "fr", "francais": "fr", "portuguese": "pt-BR", "português": "pt-BR", "russian": "ru", "русский": "ru"}
	if c, ok := names[s]; ok {
		return c
	}
	base := s
	if i := strings.IndexByte(s, '-'); i > 0 {
		base = s[:i]
	}
	for _, l := range Languages {
		if strings.EqualFold(l.Code, base) {
			return l.Code
		}
	}
	return "en"
}

// Detect finds the user's language: CATUDE_LANG, then the POSIX locale variables, then the operating
// system's preferred interface languages.
func Detect() string {
	if v := os.Getenv("CATUDE_LANG"); v != "" {
		return Match(v)
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(k); v != "" && v != "C" && v != "POSIX" && !strings.HasPrefix(v, "C.") {
			return Match(v)
		}
	}
	for _, l := range systemLanguages() {
		if c := Match(l); c != "en" || strings.HasPrefix(strings.ToLower(l), "en") {
			return c
		}
	}
	return "en"
}

// Catalog returns the messages of a language (for tests and tools).
func Catalog(code string) (map[string]string, error) {
	m := map[string]string{}
	b, err := files.ReadFile("locales/" + code + ".json")
	if err != nil {
		return nil, err
	}
	return m, json.Unmarshal(b, &m)
}
