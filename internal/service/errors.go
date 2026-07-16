package service

import "errors"

// Service-level sentinel errors. These are distinct from the
// entity.ErrNotFound / entity.ErrConflict returned by repositories:
// those describe *storage* outcomes ("no row"), these describe
// *business* outcomes ("these credentials are wrong"). A handler
// checks these with errors.Is to decide the HTTP status, the same
// pattern used for entity errors.
var (
	// ErrInvalidCredentials covers both "no such email" and "wrong
	// password". Deliberately the same error for both — returning a
	// different message for "email not found" vs "wrong password"
	// tells an attacker which emails are registered.
	ErrInvalidCredentials = errors.New("invalid email or password")

	ErrAccountBlocked      = errors.New("account has been blocked")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")

	// ErrIncorrectPassword is returned by ChangePassword when the
	// supplied "current password" doesn't match. Distinct from
	// ErrInvalidCredentials: this happens to an already-authenticated
	// user (they hold a valid access token), so there's no email-
	// enumeration concern in naming it precisely.
	ErrIncorrectPassword = errors.New("current password is incorrect")
)
