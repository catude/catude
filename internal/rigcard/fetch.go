package rigcard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Limits for remote cards.
const (
	MaxBytes     = 256 << 10
	FetchTimeout = 30 * time.Second
	MaxRedirects = 3
)

// Candidates turns a user input into the URLs to try, in order (discovery rules of docs/rigcard.md):
//
//	gh:owner/repo[@ref]   → raw.githubusercontent.com/owner/repo/<ref|HEAD>/rigcard.json
//	host/path             → https://host/path (JSON or HTML with <link rel=alternate>), then https://host/path.json
//	host                  → https://host/.well-known/rigcard.json
func Candidates(in string) ([]string, error) {
	in = strings.TrimSpace(in)
	if strings.HasPrefix(in, "gh:") {
		spec := strings.TrimPrefix(in, "gh:")
		ref := "HEAD"
		if i := strings.LastIndex(spec, "@"); i > 0 {
			spec, ref = spec[:i], spec[i+1:]
		}
		if !regexp.MustCompile(`^[\w.-]+/[\w.-]+$`).MatchString(spec) || !regexp.MustCompile(`^[\w./-]+$`).MatchString(ref) {
			return nil, fmt.Errorf("invalid gh: reference %q", in)
		}
		return []string{"https://raw.githubusercontent.com/" + spec + "/" + ref + "/rigcard.json"}, nil
	}
	if strings.HasPrefix(in, "http://") {
		return nil, errors.New("plain http is not allowed: use https")
	}
	raw := in
	if !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("not a URL: %q", in)
	}
	u.Fragment = ""
	p := strings.TrimSuffix(u.Path, "/")
	if p == "" {
		return []string{"https://" + u.Host + "/.well-known/rigcard.json"}, nil
	}
	if strings.HasSuffix(p, ".json") {
		return []string{u.String()}, nil
	}
	base := "https://" + u.Host + p
	return []string{base, base + ".json"}, nil
}

// Load reads a card from a local file or discovers it on the web.
func Load(ctx context.Context, in string) (Card, []byte, string, error) {
	if fi, err := os.Stat(in); err == nil && !fi.IsDir() {
		if fi.Size() > MaxBytes {
			return Card{}, nil, in, fmt.Errorf("card larger than %d KiB", MaxBytes>>10)
		}
		b, err := os.ReadFile(in)
		if err != nil {
			return Card{}, nil, in, err
		}
		c, err := Decode(b)
		return c, b, in, err
	}
	urls, err := Candidates(in)
	if err != nil {
		return Card{}, nil, "", err
	}
	client := safeClient()
	if testClient != nil {
		client = testClient
	}
	var errs []string // every attempt is reported: the first refusal is usually the informative one
	fail := func(err error) { errs = append(errs, err.Error()) }
	followed := false
	for i := 0; i < len(urls); i++ {
		b, ctype, err := get(ctx, client, urls[i])
		if err != nil {
			fail(err)
			continue
		}
		if strings.Contains(ctype, "html") {
			if alt := alternateLink(string(b), urls[i]); alt != "" && !followed && alt != urls[i] {
				followed = true // one hop: a page that links to itself or to another page ends here
				urls = append(urls[:i+1], append([]string{alt}, urls[i+1:]...)...)
			}
			fail(fmt.Errorf("%s is a web page without a Rigcard link", urls[i]))
			continue
		}
		c, err := Decode(b)
		if err != nil {
			fail(fmt.Errorf("%s: %w", urls[i], err))
			continue
		}
		return c, b, urls[i], nil
	}
	return Card{}, nil, "", fmt.Errorf("no Rigcard found for %q: %s", in, strings.Join(errs, "; "))
}

// Origin says where a card came from, so the files of a bundle can be found next to it.
type Origin struct {
	Kind string // file | github | https
	Dir  string // file: folder that holds the card
	Repo string // github: owner/repo
	Ref  string // github: commit, tag or branch (HEAD when not given)
	Base string // github: folder of the card inside the repository ("" = root)
}

var reRawGitHub = regexp.MustCompile(`^https://raw\.githubusercontent\.com/([\w.-]+/[\w.-]+)/([^/]+)/(.*)$`)

// OriginOf derives the origin from what the user typed and where the card was finally read from.
func OriginOf(input, from string) Origin {
	if !strings.HasPrefix(from, "https://") {
		abs, err := filepath.Abs(from)
		if err != nil {
			abs = from
		}
		return Origin{Kind: "file", Dir: filepath.Dir(abs)}
	}
	if m := reRawGitHub.FindStringSubmatch(from); m != nil {
		base := ""
		if i := strings.LastIndex(m[3], "/"); i >= 0 {
			base = m[3][:i]
		}
		return Origin{Kind: "github", Repo: m[1], Ref: m[2], Base: base}
	}
	return Origin{Kind: "https"}
}

// testClient replaces the network client in tests (never set in production code).
var testClient *http.Client

// Decode parses a card strictly enough to catch garbage, leniently enough for forward compatibility.
func Decode(b []byte) (Card, error) {
	var c Card
	if err := json.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("invalid JSON: %w", err)
	}
	if c.Rigcard == "" {
		return c, errors.New("not a Rigcard (missing \"rigcard\" version field)")
	}
	return c, nil
}

var reAlt = regexp.MustCompile(`(?is)<link[^>]+rel=["']alternate["'][^>]*>`)
var reHref = regexp.MustCompile(`(?i)href=["']([^"']+)["']`)

// alternateLink finds <link rel="alternate" type="application/vnd.rigcard+json" href="…"> (https only).
func alternateLink(html, base string) string {
	for _, tag := range reAlt.FindAllString(html, -1) {
		if !strings.Contains(strings.ToLower(tag), MediaType) {
			continue
		}
		m := reHref.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		b, err1 := url.Parse(base)
		h, err2 := url.Parse(m[1])
		if err1 != nil || err2 != nil {
			continue
		}
		r := b.ResolveReference(h)
		if r.Scheme == "https" {
			return r.String()
		}
	}
	return ""
}

func get(ctx context.Context, c *http.Client, u string) ([]byte, string, error) {
	return getN(ctx, c, u, MaxBytes, FetchTimeout)
}

func getN(ctx context.Context, c *http.Client, u string, max int64, timeout time.Duration) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", MediaType+", application/json;q=0.9, text/html;q=0.5")
	req.Header.Set("User-Agent", "catude-rigcard/0.1")
	resp, err := c.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(b)) > max {
		return nil, "", fmt.Errorf("%s: larger than %d KiB", u, max>>10)
	}
	return b, resp.Header.Get("Content-Type"), nil
}

// safeClient refuses plain http, more than MaxRedirects redirects and any connection to loopback,
// private, link-local or cloud-metadata addresses (checked on the resolved IP, so DNS tricks fail).
func safeClient() *http.Client {
	d := &net.Dialer{Timeout: 10 * time.Second, Control: nil}
	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if !PublicIP(ip.IP) {
					return nil, fmt.Errorf("refusing to connect to non-public address %s (%s)", ip.IP, host)
				}
			}
			return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	return &http.Client{Transport: tr, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= MaxRedirects {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" {
			return errors.New("redirect to non-https URL refused")
		}
		return nil
	}}
}

// PublicIP reports whether ip is a globally routable unicast address.
func PublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		switch {
		case v4[0] == 0, // "this network"
			v4[0] == 100 && v4[1]&0xC0 == 64,             // CGNAT 100.64/10
			v4[0] == 192 && v4[1] == 0 && v4[2] == 0,     // IETF protocol assignments 192.0.0/24
			v4[0] == 192 && v4[1] == 0 && v4[2] == 2,     // TEST-NET-1
			v4[0] == 198 && (v4[1] == 18 || v4[1] == 19), // benchmarking 198.18/15
			v4[0] == 198 && v4[1] == 51 && v4[2] == 100,  // TEST-NET-2
			v4[0] == 203 && v4[1] == 0 && v4[2] == 113,   // TEST-NET-3
			v4[0] >= 240: // reserved and broadcast
			return false
		}
		return true
	}
	// IPv6 forms that embed or stand for something else
	if len(ip) == net.IPv6len {
		switch {
		case ip[0] == 0x00 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b: // NAT64 64:ff9b::/96 → check the embedded IPv4
			return PublicIP(net.IPv4(ip[12], ip[13], ip[14], ip[15]))
		case ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8: // documentation 2001:db8::/32
			return false
		case ip[0] == 0x20 && ip[1] == 0x02: // 6to4 2002::/16 embeds an IPv4 address
			return PublicIP(net.IPv4(ip[2], ip[3], ip[4], ip[5]))
		}
	}
	return true
}

// Fetch downloads an https URL with the same protections as cards (public addresses only, https redirects,
// size and time limits). Used to fetch the artifacts a card points to.
func Fetch(ctx context.Context, u string, max int64, timeout time.Duration) ([]byte, error) {
	if !strings.HasPrefix(u, "https://") {
		return nil, fmt.Errorf("refusing non-https URL %s", u)
	}
	b, _, err := getN(ctx, safeClient(), u, max, timeout)
	return b, err
}
