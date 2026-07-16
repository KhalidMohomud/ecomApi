package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/domain/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// This file deliberately implements two different kinds of token
// with two different designs:
//
//   - Access tokens are JWTs: self-contained, signed, and stateless.
//     A handler can verify one using only the secret key, with no
//     database lookup — that's what makes them fast to check on
//     every request. The cost is that a JWT is valid until it
//     expires, full stop; there is no way to revoke one early short
//     of maintaining a blocklist, which erases the whole point of
//     being stateless. That's why access tokens are kept short-lived
//     (15 minutes by default).
//
//   - Refresh tokens are opaque random strings with a matching row
//     in the refresh_tokens table (see entity.RefreshToken). Because
//     each one is looked up in the database, it CAN be revoked
//     instantly — on logout, password change, or an admin blocking
//     the account. They're long-lived (30 days by default) precisely
//     because they're revocable; a JWT with a 30-day expiry that
//     leaked would be usable for 30 days no matter what you did.

// AccessTokenClaims is the payload encoded into every access token.
// It embeds jwt.RegisteredClaims for the standard fields (exp, iat,
// iss, sub) and adds the one piece of app-specific data every
// protected handler needs: the user's role, so authorization
// middleware can check it without a database round trip.
type AccessTokenClaims struct {
	Role entity.Role `json:"role"`
	jwt.RegisteredClaims
}

// ErrInvalidToken is returned by ParseAccessToken for any failure
// mode — expired, wrong signature, malformed — so callers can react
// uniformly ("reject the request") without inspecting which.
var ErrInvalidToken = errors.New("invalid or expired token")

// TokenManager issues and validates access tokens. It's constructed
// once in main.go with the app's JWT secret and injected into
// AuthService — nothing reaches for a package-level secret, which
// keeps this testable with any key a test wants to use.
type TokenManager struct {
	secret   []byte
	accessTTL time.Duration
}

func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), accessTTL: accessTTL}
}

// AccessTTL exposes the configured access-token lifetime so callers
// (the auth service, when building an AuthResponse) can report
// "expires_in" to the client without duplicating the duration.
func (tm *TokenManager) AccessTTL() time.Duration {
	return tm.accessTTL
}

// GenerateAccessToken creates a signed JWT for the given user.
func (tm *TokenManager) GenerateAccessToken(userID uuid.UUID, role entity.Role) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessTTL)),
			Issuer:    "ecomApi",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken verifies a token's signature and expiry and
// returns its claims. It rejects any token not signed with HS256 —
// without that check, a JWT library is vulnerable to an attacker
// crafting a token with alg:"none" or switching algorithms to trick
// the verifier into skipping the signature check entirely.
func (tm *TokenManager) ParseAccessToken(tokenString string) (*AccessTokenClaims, error) {
	claims := &AccessTokenClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return tm.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GenerateRefreshToken creates a new opaque refresh token. It
// returns two values: raw is what's sent to the client (and never
// stored anywhere), hash is what's persisted in the database so a
// later request can be matched back to this token without the
// database ever holding a usable credential.
func GenerateRefreshToken() (raw string, hash string, err error) {
	buf := make([]byte, 32) // 256 bits of entropy
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashToken(raw), nil
}

// HashToken deterministically hashes a raw refresh token so it can
// be looked up by hash. SHA-256 (not bcrypt) is correct here: bcrypt
// is deliberately slow to resist brute-forcing a low-entropy human
// password; a refresh token is already 256 bits of random data, so
// there's nothing to brute-force — we just need a fast, deterministic
// lookup key.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
