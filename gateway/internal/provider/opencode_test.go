package provider

import (
	"regexp"
	"testing"
)

func TestOpenCodeSessionIDFormat(t *testing.T) {
	pattern := regexp.MustCompile(`^ses_[0-9a-f]{12}[0-9A-Za-z]{14}$`)
	for range 50 {
		id := newOpenCodeSessionID()
		if !pattern.MatchString(id) {
			t.Fatalf("unexpected session id %q", id)
		}
	}
}
