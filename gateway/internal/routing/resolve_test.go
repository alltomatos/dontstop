package routing_test

import (
	"errors"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/routing"
)

type fakeDecrypter struct {
	plaintext []byte
	err       error
}

func (d fakeDecrypter) Decrypt([]byte) ([]byte, error) {
	return d.plaintext, d.err
}

func TestResolveAccountDecryptsCredentialAndPreservesEndpoint(t *testing.T) {
	account, err := routing.ResolveAccount(routing.Candidate{
		AccountID:        "account-1",
		BaseURL:          "https://provider.example/v1",
		CredentialCipher: []byte("ciphertext"),
	}, fakeDecrypter{plaintext: []byte("sk-secret")})
	if err != nil {
		t.Fatalf("resolve account: %v", err)
	}
	if account.ID != "account-1" || account.BaseURL != "https://provider.example/v1" || account.APIKey != "sk-secret" {
		t.Fatalf("unexpected account: %+v", account)
	}
}

func TestResolveAccountFailsWhenDecryptionFails(t *testing.T) {
	_, err := routing.ResolveAccount(routing.Candidate{AccountID: "account-1"}, fakeDecrypter{err: errors.New("bad key")})
	if err == nil {
		t.Fatal("expected decryption error")
	}
}
