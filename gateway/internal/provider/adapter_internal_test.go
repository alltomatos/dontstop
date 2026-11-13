package provider

import "testing"

func TestLooksLikeBan(t *testing.T) {
	banSignals := []string{
		`{"error":"your IP address is banned"}`,
		"Access blocked from this region",
		"suspicious activity detected",
	}
	for _, body := range banSignals {
		if !looksLikeBan([]byte(body)) {
			t.Fatalf("expected a ban signal in %q", body)
		}
	}
	notBans := []string{
		`{"error":"invalid api key"}`,
		"rate limit exceeded",
		`{"error":"model not found"}`,
	}
	for _, body := range notBans {
		if looksLikeBan([]byte(body)) {
			t.Fatalf("unexpected ban signal in %q", body)
		}
	}
}
