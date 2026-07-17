package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Minimal HS256 JWT implementation using only the standard library, so the
// project has zero third-party dependencies and builds reliably offline.

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// Claims are the JWT payload fields used by this service.
type Claims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func b64Encode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// GenerateToken creates a signed JWT for the given user, valid for ttl.
func GenerateToken(secret, userID, email string, ttl time.Duration) (string, error) {
	h := header{Alg: "HS256", Typ: "JWT"}
	now := time.Now()
	c := Claims{
		UserID:    userID,
		Email:     email,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
	}

	hBytes, err := json.Marshal(h)
	if err != nil {
		return "", err
	}
	cBytes, err := json.Marshal(c)
	if err != nil {
		return "", err
	}

	unsigned := b64Encode(hBytes) + "." + b64Encode(cBytes)
	sig := sign(secret, unsigned)
	return unsigned + "." + sig, nil
}

// ParseToken validates the token signature and expiry, returning its claims.
func ParseToken(secret, token string) (*Claims, error) {
	var parts [3]string
	start := 0
	idx := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			if idx > 2 {
				return nil, ErrInvalidToken
			}
			parts[idx] = token[start:i]
			idx++
			start = i + 1
		}
	}
	if idx != 2 {
		return nil, ErrInvalidToken
	}
	parts[2] = token[start:]

	unsigned := parts[0] + "." + parts[1]
	expectedSig := sign(secret, unsigned)
	if subtle.ConstantTimeCompare([]byte(expectedSig), []byte(parts[2])) != 1 {
		return nil, ErrInvalidToken
	}

	cBytes, err := b64Decode(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(cBytes, &c); err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() > c.ExpiresAt {
		return nil, ErrExpiredToken
	}
	return &c, nil
}

func sign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return b64Encode(mac.Sum(nil))
}

// ExtractBearerToken pulls the token out of an `Authorization: Bearer <token>` header value.
func ExtractBearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
		return "", fmt.Errorf("missing or malformed Authorization header")
	}
	return header[len(prefix):], nil
}
