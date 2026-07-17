package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2id recommended defaults for interactive use (64MB memory, 1 iteration, 4 threads)
	hashTime    = 1
	hashMemory  = 32 * 1024
	hashThreads = 4
	hashKeyLen  = 32
	hashSaltLen = 16
)

// HashPassword hashes a password using Argon2id.
// The output is a self-contained encoded string: $argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, hashSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, hashTime, hashMemory, hashThreads, hashKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, hashMemory, hashTime, hashThreads, b64Salt, b64Hash), nil
}

// CheckPassword verifies a password against an Argon2id encoded hash.
// Returns true if the password matches.
func CheckPassword(password, encodedHash string) bool {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false
	}

	computed := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(hash)))
	return constantTimeCompare(computed, hash)
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
}

// decodeHash parses an Argon2id encoded hash string.
// Format: $argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
func decodeHash(encoded string) (params argon2Params, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return params, nil, nil, errors.New("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return params, nil, nil, errors.New("unsupported algorithm: " + parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return params, nil, nil, fmt.Errorf("invalid version: %w", err)
	}
	if version != argon2.Version {
		return params, nil, nil, fmt.Errorf("unexpected argon2 version: %d", version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.time, &params.threads); err != nil {
		return params, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, fmt.Errorf("invalid salt encoding: %w", err)
	}

	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params, nil, nil, fmt.Errorf("invalid hash encoding: %w", err)
	}

	return params, salt, hash, nil
}

// constantTimeCompare compares two byte slices in constant time.
func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := byte(0)
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
