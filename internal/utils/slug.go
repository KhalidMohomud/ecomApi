package utils

import (
	"regexp"
	"strings"
)

// apostrophes are dropped entirely rather than turned into a
// separator — "Men's" should slugify to "mens", not "men-s". Any
// other punctuation (spaces, "!", "&", ...) becomes a hyphen via
// nonSlugChars below, but an apostrophe inside a word reads as part
// of the word, so it gets special treatment first.
var apostrophes = regexp.MustCompile(`['’]`)

// nonSlugChars matches any run of characters that aren't lowercase
// letters or digits — spaces, punctuation, accented characters,
// emoji, anything else. Compiled once at package init instead of
// inside Slugify, since regexp.MustCompile is relatively expensive
// and this function runs on every category/brand/product write.
var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns arbitrary text into a URL-safe, lowercase,
// hyphen-separated identifier: "Men's Running Shoes!" -> "mens-running-shoes".
//
// It's used two ways by the services that call it: to generate a
// slug automatically when a client doesn't supply one, and to
// normalize a slug the client DID supply — so "Shoes" and "shoes"
// can't both slip past the database's unique index as
// case-different-but-functionally-identical values.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = apostrophes.ReplaceAllString(s, "")
	s = nonSlugChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
