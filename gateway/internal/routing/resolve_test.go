package routing_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
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

func TestResolveAccountIgnoresAssignedProxyForOff(t *testing.T) {
	account, err := routing.ResolveAccount(routing.Candidate{
		AccountID:        "account-1",
		CredentialCipher: []byte("ciphertext"),
		ProxyMode:        "off",
		ProxyAddress:     "203.0.113.10",
		ProxyPort:        8080,
		ProxyHealthy:     true,
	}, fakeDecrypter{plaintext: []byte("sk-secret")})
	if err != nil {
		t.Fatalf("resolve account: %v", err)
	}
	if account.ProxyURL != "" {
		t.Fatalf("off must never acquire a proxy, got %q", account.ProxyURL)
	}
}

func TestResolveAccountFallsBackDirectForPreferred(t *testing.T) {
	account, err := routing.ResolveAccount(routing.Candidate{
		AccountID:        "account-1",
		CredentialCipher: []byte("ciphertext"),
		ProxyMode:        "preferred",
		ProxyAddress:     "203.0.113.10",
		ProxyPort:        8080,
		ProxyHealthy:     false,
	}, fakeDecrypter{plaintext: []byte("sk-secret")})
	if err != nil {
		t.Fatalf("resolve account: %v", err)
	}
	if account.ProxyURL != "" {
		t.Fatalf("preferred must fall back direct, got %q", account.ProxyURL)
	}
}

func TestResolveAccountFailsClosedForRequired(t *testing.T) {
	_, err := routing.ResolveAccount(routing.Candidate{
		AccountID:        "account-1",
		CredentialCipher: []byte("ciphertext"),
		ProxyMode:        "required",
		ProxyAddress:     "203.0.113.10",
		ProxyPort:        8080,
		ProxyHealthy:     false,
	}, fakeDecrypter{plaintext: []byte("sk-secret")})
	if !errors.Is(err, provider.ErrProxyRequired) {
		t.Fatalf("expected ErrProxyRequired, got %v", err)
	}
}

func TestResolveAccountUsesHealthyProxy(t *testing.T) {
	account, err := routing.ResolveAccount(routing.Candidate{
		AccountID:           "account-1",
		CredentialCipher:    []byte("ciphertext"),
		ProxyMode:           "required",
		ProxyAddress:        "203.0.113.10",
		ProxyPort:           8080,
		ProxyUsernameCipher: []byte("user"),
		ProxyPasswordCipher: []byte("pass"),
		ProxyHealthy:        true,
	}, fakeDecrypter{plaintext: []byte("sk-secret")})
	if err != nil {
		t.Fatalf("resolve account: %v", err)
	}
	if !strings.HasPrefix(account.ProxyURL, "http://") || !strings.Contains(account.ProxyURL, "203.0.113.10:8080") {
		t.Fatalf("unexpected proxy URL: %q", account.ProxyURL)
	}
}
