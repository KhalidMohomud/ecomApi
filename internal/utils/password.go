package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost controls how much CPU work hashing one password takes.
// bcrypt's default is 10; we use 12. Each +1 roughly doubles the
// work, so 12 is ~4x slower than the default — deliberately, since
// that cost applies equally to an attacker brute-forcing a stolen
// hash offline. It's a trade-off against login latency (still well
// under 300ms), not a free win, which is why it's a named constant
// instead of a magic number, and why teams periodically raise it as
// hardware gets faster.
const bcryptCost = 12

// HashPassword returns a bcrypt hash of the given plaintext
// password. The returned string already encodes the algorithm,
// cost, and a random salt — that's why CheckPassword below needs no
// separate salt parameter.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the given bcrypt
// hash. It never returns the underlying error to the caller —
// bcrypt.CompareHashAndPassword returns an error for "hash doesn't
// match" and for "hash is malformed" alike, and both cases mean
// exactly one thing to a caller: reject the login.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
