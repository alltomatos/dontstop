package routing

import (
	"fmt"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

// Decrypter decrypts provider credentials held in memory.
type Decrypter interface {
	Decrypt(ciphertext []byte) ([]byte, error)
}

// ResolveAccount decrypts a candidate credential into a provider account.
func ResolveAccount(candidate Candidate, decrypter Decrypter) (provider.Account, error) {
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
	return provider.Account{
		ID:      candidate.AccountID,
		BaseURL: candidate.BaseURL,
		APIKey:  string(plaintext),
	}, nil
}
