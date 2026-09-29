package oracle

import "testing"

func TestNormalizeTransferID(t *testing.T) {
	if got := NormalizeTransferID("15,862,087,576"); got != "15862087576" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTransferIDNumeric(t *testing.T) {
	if got := ParseTransferIDNumeric("15,875,930,915"); got != 15875930915 {
		t.Fatalf("got %d", got)
	}
}

func TestNormalizeMSISDN(t *testing.T) {
	if got := NormalizeMSISDN("252639339979"); got != "252639339979" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeMSISDN("abc"); got != "" {
		t.Fatalf("expected empty got %q", got)
	}
}
