package routing

import (
	"fmt"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

// Decrypter decrypts provider and proxy credentials held in memory.
type Decrypter interface {
	Decrypt(ciphertext []byte) ([]byte, error)
}

// ResolveCredentials decrypts the provider credential without resolving a proxy.
// Credentials are never included in errors.
func ResolveCredentials(candidate Candidate, decrypter Decrypter) (provider.Account, error) {
	if decrypter == nil {
		return provider.Account{}, fmt.Errorf("credential decrypter is required")
	}
	if len(candidate.CredentialCipher) == 0 {
		return provider.Account{}, fmt.Errorf("provider account has no credential")
	}
	plaintext, err := decrypter.Decrypt(candidate.CredentialCipher)
	if err != nil {
		return provider.Account{}, fmt.Errorf("decrypt provider credential: %w", err)
	}
	if len(plaintext) == 0 {
		return provider.Account{}, fmt.Errorf("provider credential is empty")
	}
	return provider.Account{ID: candidate.AccountID, BaseURL: candidate.BaseURL, APIKey: string(plaintext)}, nil
}

// ResolveAccount decrypts credentials and resolves the candidate proxy fields.
func ResolveAccount(candidate Candidate, decrypter Decrypter) (provider.Account, error) {
	account, err := ResolveCredentials(candidate, decrypter)
	if err != nil {
		return provider.Account{}, err
	}
	mode := provider.NormalizeProxyMode(candidate.ProxyMode)
	useProxy, err := provider.SelectProxy(mode, candidate.ProxyHealthy)
	if err != nil {
		return provider.Account{}, err
	}
	if !useProxy {
		return account, nil
	}

	username := ""
	password := ""
	if len(candidate.ProxyUsernameCipher) > 0 {
		plain, err := decrypter.Decrypt(candidate.ProxyUsernameCipher)
		if err != nil {
			return provider.Account{}, fmt.Errorf("decrypt proxy username: %w", err)
		}
		username = string(plain)
	}
	if len(candidate.ProxyPasswordCipher) > 0 {
		plain, err := decrypter.Decrypt(candidate.ProxyPasswordCipher)
		if err != nil {
			return provider.Account{}, fmt.Errorf("decrypt proxy password: %w", err)
		}
		password = string(plain)
	}
	proxyURL, err := provider.BuildProxyURL(candidate.ProxyAddress, candidate.ProxyPort, username, password)
	if err != nil {
		return provider.Account{}, fmt.Errorf("build proxy URL: %w", err)
	}
	account.ProxyURL = proxyURL
	return account, nil
}
