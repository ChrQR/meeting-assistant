package client

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SessionClaims is the payload of our own session JWT. We issue this token
// ourselves (rather than passing the provider's token through) so session
// lifetime and claims stay under our control.
// SessionTTL is how long a session stays valid.
const SessionTTL = 7 * 24 * time.Hour

type SessionClaims struct {
	UserID uuid.UUID `json:"sub"`
	Email  string    `json:"email"`
	Issued time.Time `json:"iat"`
	Expiry time.Time `json:"exp"`
}

type SessionSigner interface {
	Sign(claims SessionClaims) (string, error)
	Verify(token string) (SessionClaims, error)
}

type sessionSigner struct {
	secret []byte
	ttl    time.Duration
}

func NewSessionSigner(secret string, ttl time.Duration) SessionSigner {
	return &sessionSigner{secret: []byte(secret), ttl: ttl}
}

func (s *sessionSigner) Sign(claims SessionClaims) (string, error) {
	now := time.Now()
	claims.Issued = now
	claims.Expiry = now.Add(s.ttl)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signature := s.sign(header + "." + payload)
	return header + "." + payload + "." + signature, nil
}

func (s *sessionSigner) Verify(token string) (SessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return SessionClaims{}, errors.New("malformed token")
	}

	expected := s.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return SessionClaims{}, errors.New("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return SessionClaims{}, fmt.Errorf("decode payload: %w", err)
	}

	var claims SessionClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return SessionClaims{}, fmt.Errorf("decode claims: %w", err)
	}
	if time.Now().After(claims.Expiry) {
		return SessionClaims{}, errors.New("token expired")
	}
	return claims, nil
}

func (s *sessionSigner) sign(input string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
