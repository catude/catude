package accounts

import "testing"

const sample = `{"schemaVersion":1,"activeAccountNumber":2,"accounts":[
 {"number":1,"email":"ana@example.com","organizationName":"Acme","isOrganization":true,"active":false,"usageStatus":"ok","loginExpiresAt":"2026-12-01T00:00:00Z",
  "lastGoodUsage":{"fiveHour":{"pct":12.5,"resetsAt":"x"},"sevenDay":{"pct":87,"projectedExhaustionAt":"2026-11-19T10:00:00Z","willLastToReset":false}}},
 {"number":2,"email":"bo@example.com","organizationName":"","isOrganization":false,"active":true,"usageStatus":"ok","lastGoodUsage":null}]}`

func TestParse(t *testing.T) {
	a, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || !a[1].Active || a[0].OrganizationName != "Acme" {
		t.Fatalf("accounts: %+v", a)
	}
	if a[0].SevenDay() == nil || a[0].SevenDay().Pct != 87 || *a[0].SevenDay().WillLastToReset {
		t.Fatalf("seven day: %+v", a[0].SevenDay())
	}
	if a[1].FiveHour() != nil {
		t.Fatal("missing usage must be nil, not zero")
	}
}

func TestSchemaAndVersions(t *testing.T) {
	if _, err := Parse([]byte(`{"schemaVersion":2,"accounts":[]}`)); err == nil {
		t.Fatal("unknown schema must be reported")
	}
	if _, err := Parse([]byte(`Account 1: ana@example.com`)); err == nil {
		t.Fatal("non-JSON output (old claude-swap) must be an error")
	}
	if !Less("0.27.2", "0.30.0") || Less("1.1.0", "0.30.0") || Less("0.30.0", "0.30.0") {
		t.Fatal("version comparison")
	}
	if w := (Switcher{Version: "0.27.2"}).Warnings(); len(w) != 2 {
		t.Fatalf("old version must warn about #516: %v", w)
	}
	if MaskEmail("ana@example.com") != "a••••@example.com" {
		t.Fatal(MaskEmail("ana@example.com"))
	}
}
