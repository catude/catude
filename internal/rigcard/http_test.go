package rigcard

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The whole discovery protocol against a real HTTPS server (the test client only replaces the
// "public addresses only" dialer, which cannot reach a test server on localhost).
func TestDiscoveryOverHTTPS(t *testing.T) {
	const card = `{"rigcard":"0.1","summary":"served card","skills":[{"name":"tdd","source":{"type":"bundle","path":"skills/tdd"},"sha256":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `"}]}`
	mux := http.NewServeMux()
	var accept string
	mux.HandleFunc("/rig.json", func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", MediaType)
		fmt.Fprint(w, card)
	})
	mux.HandleFunc("/rig", func(w http.ResponseWriter, r *http.Request) { // a human page that links the card
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><head><link rel="alternate" type="application/vnd.rigcard+json" href="/rig.json"></head><body>my rig</body></html>`)
	})
	mux.HandleFunc("/plain", func(w http.ResponseWriter, r *http.Request) { // a page without a link: falls back to /plain.json
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>nothing here</body></html>")
	})
	mux.HandleFunc("/plain.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, card) })
	mux.HandleFunc("/.well-known/rigcard.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, card) })
	mux.HandleFunc("/moved", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/rig.json", http.StatusFound) })
	mux.HandleFunc("/downgrade", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://"+r.Host+"/rig.json", http.StatusFound)
	})
	mux.HandleFunc("/loop", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/loop", http.StatusFound) })
	mux.HandleFunc("/huge.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"rigcard":"0.1","summary":"`+strings.Repeat("x", MaxBytes+10)+`"}`)
	})
	mux.HandleFunc("/broken.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"rigcard":`) })
	mux.HandleFunc("/notacard.json", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"name":"package"}`) })
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()

	c := srv.Client()
	c.CheckRedirect = safeClient().CheckRedirect // keep the real redirect policy
	testClient = c
	defer func() { testClient = nil }()
	host := strings.TrimPrefix(srv.URL, "https://")

	for _, in := range []string{host + "/rig.json", host + "/rig", host + "/plain", host, host + "/moved"} {
		got, raw, from, err := Load(context.Background(), in)
		if err != nil || got.Summary != "served card" || len(raw) == 0 {
			t.Errorf("%s: %v (from %s)", in, err, from)
			continue
		}
		if o := OriginOf(in, from); o.Kind != "https" {
			t.Errorf("%s: origin %+v", in, o)
		}
	}
	if !strings.Contains(accept, MediaType) {
		t.Errorf("the client must ask for the Rigcard media type, sent %q", accept)
	}
	for in, want := range map[string]string{
		host + "/downgrade":     "non-https",
		host + "/loop":          "redirects",
		host + "/huge.json":     "larger than",
		host + "/broken.json":   "invalid JSON",
		host + "/notacard.json": "not a Rigcard",
		host + "/missing.json":  "HTTP 404",
	} {
		if _, _, _, err := Load(context.Background(), in); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want an error containing %q, got %v", in, want, err)
		}
	}
}

// Without the test client the real guard applies: nothing on this machine or a private network is reachable.
func TestLoadRefusesInternalAddresses(t *testing.T) {
	for _, in := range []string{"localhost/rig", "127.0.0.1/rig.json", "169.254.169.254/latest/meta-data", "[::1]/rig", "http://example.com/rig"} {
		if _, _, _, err := Load(context.Background(), in); err == nil {
			t.Errorf("%s must be refused", in)
		}
	}
}

func TestOriginOf(t *testing.T) {
	o := OriginOf("gh:ada/rigs@v2", "https://raw.githubusercontent.com/ada/rigs/v2/rigcard.json")
	if o.Kind != "github" || o.Repo != "ada/rigs" || o.Ref != "v2" || o.Base != "" {
		t.Errorf("%+v", o)
	}
	o = OriginOf("x", "https://raw.githubusercontent.com/ada/rigs/abc123/profiles/game/rigcard.json")
	if o.Base != "profiles/game" || o.Ref != "abc123" {
		t.Errorf("%+v", o)
	}
	if o := OriginOf("./d/rigcard.json", "d/rigcard.json"); o.Kind != "file" || !strings.HasSuffix(strings.ReplaceAll(o.Dir, `\`, "/"), "/d") {
		t.Errorf("%+v", o)
	}
}
