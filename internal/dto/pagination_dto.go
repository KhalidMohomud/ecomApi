package dto

// PaginationQuery is bound from URL query parameters (?page=2&page_size=10),
// not a JSON body — list endpoints are GET requests, so there's no
// body to bind. c.ShouldBindQuery reads it instead of c.ShouldBindJSON.
//
// The `,default=` part of the form tag is gin/binding syntax: if the
// query parameter is absent, gin fills in that value before running
// the binding rules below it, so a request with no ?page at all still
// ends up with Page=1, PageSize=20 rather than the zero values 0, 0.
type PaginationQuery struct {
	Page     int `form:"page,default=1" binding:"omitempty,min=1"`
	PageSize int `form:"page_size,default=20" binding:"omitempty,min=1,max=100"`
}

// Offset converts a 1-indexed page number into the 0-indexed offset
// every repository's List method expects.
func (q PaginationQuery) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// PaginatedResponse wraps any list of items with the paging metadata
// a client needs to render "page 2 of 7" and decide whether to
// request more.
//
// [T any] is a Go generic type parameter — introduced in Go 1.18.
// Without it, we'd need a separate UserListResponse, ProductListResponse,
// OrderListResponse, etc., each identical except for the Items field's
// type. PaginatedResponse[dto.UserResponse] and, later,
// PaginatedResponse[dto.ProductResponse] both reuse this one
// definition; the compiler generates the specific version for each T
// used, so there's no runtime cost to the flexibility.
type PaginatedResponse[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginatedResponse computes TotalPages from total/pageSize so
// every caller doesn't have to hand-roll the ceiling-division math
// (and risk getting the off-by-one wrong).
func NewPaginatedResponse[T any](items []T, total int64, page, pageSize int) PaginatedResponse[T] {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	return PaginatedResponse[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
