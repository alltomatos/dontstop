package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const keyLength = 32

// Box encrypts and decrypts provider credentials with AES-256-GCM.
type Box struct {
	aead cipher.AEAD
}

// NewBox creates a Box from a 32-byte key.
func NewBox(key []byte) (*Box, error) {
	return newBox(key)
}

// NewBoxFromBase64 creates a Box from a base64-encoded 32-byte key.
func NewBoxFromBase64(encoded string) (*Box, error) {
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode master key: %w", err)
	}
	return newBox(key)
}

func newBox(key []byte) (*Box, error) {
	if len(key) != keyLength {
		return nil, fmt.Errorf("master key must be %d bytes, got %d", keyLength, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Encrypt returns version || nonce || ciphertext.
func (b *Box) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := b.aead.Seal(nil, nonce, plaintext, nil)
	result := make([]byte, 0, 1+len(nonce)+len(ciphertext))
	result = append(result, 1)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// Decrypt validates the version, nonce and GCM authentication tag.
func (b *Box) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := b.aead.NonceSize()
	if len(ciphertext) < 1+nonceSize || ciphertext[0] != 1 {
		return nil, fmt.Errorf("invalid ciphertext envelope")
	}
	nonce := ciphertext[1 : 1+nonceSize]
	plaintext, err := b.aead.Open(nil, nonce, ciphertext[1+nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential: %w", err)
	}
	return plaintext, nil
}
