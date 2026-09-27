package oracle

import "testing"

func TestParseConnString(t *testing.T) {
	host, port, svc, err := parseConnString("db.example.com:1521/ORCL")
	if err != nil {
		t.Fatal(err)
	}
	if host != "db.example.com" || port != 1521 || svc != "ORCL" {
		t.Fatalf("got %q %d %q", host, port, svc)
	}
}

func TestBuildDSN_oracleURLPassthrough(t *testing.T) {
	raw := "oracle://user:pass@host:1521/service"
	got, err := buildDSN("ignored", "ignored", raw)
	if err != nil || got != raw {
		t.Fatalf("got %q err=%v", got, err)
	}
}
