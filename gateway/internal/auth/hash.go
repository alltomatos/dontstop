package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltLength  = 16
	argonKeyLength   = 32
)

// HashKey derives an Argon2id PHC string for an API key.
func HashKey(key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("API key is empty")
	}
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(key), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyKey checks an API key against a PHC-encoded Argon2id hash.
func VerifyKey(encoded, key string) bool {
	params, salt, expected, err := parsePHC(encoded)
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(key), salt, params.iterations, params.memory, params.parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parsePHC(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argonParams{}, nil, nil, fmt.Errorf("invalid PHC hash")
	}

	params := argonParams{}
	for _, item := range strings.Split(parts[3], ",") {
		name, value, found := strings.Cut(item, "=")
		if !found {
			return argonParams{}, nil, nil, fmt.Errorf("invalid PHC parameter")
		}
		switch name {
		case "m":
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return argonParams{}, nil, nil, fmt.Errorf("invalid memory parameter")
			}
			params.memory = uint32(parsed)
		case "t":
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return argonParams{}, nil, nil, fmt.Errorf("invalid iterations parameter")
			}
			params.iterations = uint32(parsed)
		case "p":
			parsed, err := strconv.ParseUint(value, 10, 8)
			if err != nil {
				return argonParams{}, nil, nil, fmt.Errorf("invalid parallelism parameter")
			}
			params.parallelism = uint8(parsed)
		default:
			return argonParams{}, nil, nil, fmt.Errorf("unknown PHC parameter")
		}
	}
	if params.memory == 0 || params.iterations == 0 || params.parallelism == 0 {
		return argonParams{}, nil, nil, fmt.Errorf("incomplete PHC parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return argonParams{}, nil, nil, fmt.Errorf("invalid PHC salt")
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return argonParams{}, nil, nil, fmt.Errorf("invalid PHC hash")
	}
	return params, salt, hash, nil
}
