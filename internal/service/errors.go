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

	// ErrCannotModifySelf guards the admin block/delete endpoints: an
	// admin acting on their own account (via the admin API, not the
	// self-service /users/me endpoints) is almost always a mistake —
	// most commonly a script iterating "all users" without excluding
	// the caller — and blocking or deleting your own only admin
	// account is a lockout with no way back through the API.
	ErrCannotModifySelf = errors.New("cannot perform this action on your own account")

	// ErrInvalidParentCategory covers both cases the service checks
	// before writing a category: the given parent_id doesn't exist,
	// or it's the category's own ID (a category cannot be its own
	// parent). Both are "the request doesn't make sense," not "the
	// database rejected something," so this is a service error, not
	// a repository one.
	ErrInvalidParentCategory = errors.New("invalid parent category")
)
