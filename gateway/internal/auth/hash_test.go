package auth_test

import (
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/auth"
)

func TestHashKeyVerifiesTheOriginalSecret(t *testing.T) {
	const key = "sk-ds-0123456789abcdef"

	encoded, err := auth.HashKey(key)
	if err != nil {
		t.Fatalf("hash key: %v", err)
	}
	if encoded == key {
		t.Fatal("hash must not equal the plaintext key")
	}
	if !auth.VerifyKey(encoded, key) {
		t.Fatal("expected the original key to verify")
	}
	if auth.VerifyKey(encoded, "sk-ds-wrong") {
		t.Fatal("a different key must not verify")
	}
}

func TestVerifyKeyRejectsMalformedHash(t *testing.T) {
	if auth.VerifyKey("not-a-phc-hash", "sk-ds-0123456789abcdef") {
		t.Fatal("malformed hash must not verify")
	}
}
