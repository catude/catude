package i18n

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	for in, want := range map[string]string{
		"ja_JP.UTF-8": "ja", "ja-JP": "ja", "zh_CN.UTF-8": "zh-Hans", "zh-TW": "zh-Hant", "zh-Hant-HK": "zh-Hant",
		"zh-Hans": "zh-Hans", "pt_BR": "pt-BR", "pt-PT": "pt-BR", "es-419": "es", "es_ES.UTF-8": "es", "de-AT": "de",
		"ko_KR": "ko", "ru": "ru", "fr-CA": "fr", "en_US.UTF-8": "en", "": "en", "xx": "en", "japanese": "ja", "Español": "es",
	} {
		if got := Match(in); got != want {
			t.Errorf("Match(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFallbackAndFormat(t *testing.T) {
	defer Set("en")
	Set("en")
	if T("Sessions") != "Sessions" {
		t.Error("English is the identity")
	}
	Set("xx")
	if Lang() != "en" {
		t.Error("unknown languages fall back to English")
	}
	if got := F("%d open · %d working", 3, 1); got != "3 open · 1 working" {
		t.Errorf("F: %q", got)
	}
	if T("a message nobody translated") != "a message nobody translated" {
		t.Error("an untranslated message is shown in English, never blank")
	}
}

var rePlaceholder = regexp.MustCompile(`%[-+# 0]*\d*(\.\d+)?[sdvfqx%]`)

// keys that must stay as they are in key hints
var keyWords = []string{"enter", "esc", "space", "tab", "pgup", "pgdn", "ctrl+c", "F2", "F12"}

// hintKeys are the key names that open a segment of a key hint ("enter open · esc back"): only there is
// a word like "tab" the key and not the noun ("click a tab").
func hintKeys(id string) []string {
	if !strings.Contains(id, " · ") {
		return nil
	}
	var out []string
	for _, seg := range strings.Split(id, " · ") {
		f := strings.Fields(seg)
		if len(f) == 0 {
			continue
		}
		for _, k := range keyWords {
			if f[0] == k {
				out = append(out, k)
			}
		}
	}
	return out
}

// Every language covers every message, with the same placeholders and the same key names; names of
// products and commands are kept.
func TestCatalogsAreComplete(t *testing.T) {
	b, err := files.ReadFile("locales/messages.json")
	if err != nil {
		t.Fatal(err)
	}
	var list []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &list); err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, m := range list {
		ids[m.ID] = true
	}
	for _, l := range Languages[1:] {
		cat, err := Catalog(l.Code)
		if err != nil {
			t.Errorf("%s: %v", l.Code, err)
			continue
		}
		var missing, extra []string
		for id := range ids {
			if strings.TrimSpace(cat[id]) == "" {
				missing = append(missing, id)
			}
		}
		for id := range cat {
			if !ids[id] {
				extra = append(extra, id)
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		if len(missing) > 0 {
			t.Errorf("%s: %d messages not translated, e.g. %q", l.Code, len(missing), missing[0])
		}
		if len(extra) > 0 {
			t.Errorf("%s: %d messages no longer used, e.g. %q (run scripts/i18n-extract.mjs)", l.Code, len(extra), extra[0])
		}
		for id, tr := range cat {
			if !ids[id] {
				continue
			}
			if a, b := rePlaceholder.FindAllString(id, -1), rePlaceholder.FindAllString(tr, -1); strings.Join(a, " ") != strings.Join(b, " ") {
				t.Errorf("%s: placeholders differ in %q: %v vs %v (%q)", l.Code, id, a, b, tr)
			}
			for _, k := range hintKeys(id) {
				if !strings.Contains(tr, k) {
					t.Errorf("%s: key %q must stay untranslated in %q → %q", l.Code, k, id, tr)
				}
			}
			for _, name := range []string{"catude", "ctlite", "Claude Code", "Claude Desktop", "MCP", "Mermaid", "claude mcp list", "claude /context"} {
				if strings.Contains(id, name) && !strings.Contains(tr, name) {
					t.Errorf("%s: %q must stay in %q → %q", l.Code, name, id, tr)
				}
			}
		}
	}
}
