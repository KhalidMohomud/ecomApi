package entity

// UserStats is an aggregate, not a table row — there's no "user_stats"
// table, it's the result of a COUNT query. It lives in entity anyway
// because, like User, it's a shape both the repository and service
// layers need to agree on, and entity is the layer both already
// depend on.
type UserStats struct {
	Total   int64
	Active  int64
	Blocked int64
}
