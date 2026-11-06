package secret_test

import (
	"bytes"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/platform/secret"
)

func TestBoxRoundTripAndTamperDetection(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	box, err := secret.NewBox(key)
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	plaintext := []byte("sk-ds-super-secret")
	ciphertext, err := box.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Contains(ciphertext, plaintext) {
		t.Fatal("ciphertext must not contain plaintext")
	}
	decrypted, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("roundtrip mismatch: %q", decrypted)
	}

	wrongBox, err := secret.NewBox(bytes.Repeat([]byte{0x43}, 32))
	if err != nil {
		t.Fatalf("new wrong box: %v", err)
	}
	if _, err := wrongBox.Decrypt(ciphertext); err == nil {
		t.Fatal("wrong key must fail")
	}

	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0xff
	if _, err := box.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext must fail")
	}
}

func TestNewBoxRejectsInvalidKeyLength(t *testing.T) {
	if _, err := secret.NewBox([]byte("short")); err == nil {
		t.Fatal("expected an invalid key length error")
	}
}
