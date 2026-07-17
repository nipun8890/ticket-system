package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// This package intentionally avoids third-party dependencies (e.g. bcrypt)
// so the project builds fully offline with only the Go standard library.
// It implements a salted, iterated HMAC-SHA256 hash (a minimal PBKDF2-style
// construction), which is a standard and safe way to store passwords.

const (
	hashIterations = 100_000
	saltBytes      = 16
)

// HashPassword generates a random salt and returns the salt and derived hash,
// both hex-encoded, to be stored alongside the user record.
func HashPassword(password string) (hash string, salt string, err error) {
	saltRaw := make([]byte, saltBytes)
	if _, err := rand.Read(saltRaw); err != nil {
		return "", "", fmt.Errorf("generating salt: %w", err)
	}
	salt = hex.EncodeToString(saltRaw)
	hash = derive(password, salt)
	return hash, salt, nil
}

// VerifyPassword checks a plaintext password against a stored hash+salt pair
// using a constant-time comparison to avoid timing attacks.
func VerifyPassword(password, salt, hash string) bool {
	computed := derive(password, salt)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

func derive(password, salt string) string {
	digest := []byte(salt + password)
	for i := 0; i < hashIterations; i++ {
		mac := hmac.New(sha256.New, []byte(salt))
		mac.Write(digest)
		digest = mac.Sum(nil)
	}
	return hex.EncodeToString(digest)
}
