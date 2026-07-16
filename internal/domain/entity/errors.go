package entity

import "errors"

// Sentinel errors returned by repositories across every entity, not
// just User. A repository translates whatever its underlying store
// says (a GORM error, a Postgres error code) into one of these, so
// the service layer can write `errors.Is(err, entity.ErrNotFound)`
// without knowing or caring that GORM or Postgres are involved.
var (
	ErrNotFound = errors.New("resource not found")
	ErrConflict = errors.New("resource already exists")
)
