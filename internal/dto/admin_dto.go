package dto

// DashboardResponse summarizes platform state for the admin
// dashboard. It's user-only for now because Users is the only
// domain that exists yet — once Products/Orders ship in later steps,
// this struct grows fields like TotalOrders and RevenueToday rather
// than becoming a new endpoint, since it's still fundamentally "one
// screen of at-a-glance numbers."
type DashboardResponse struct {
	TotalUsers   int64 `json:"total_users"`
	ActiveUsers  int64 `json:"active_users"`
	BlockedUsers int64 `json:"blocked_users"`
}
