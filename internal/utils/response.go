package utils

import "github.com/gin-gonic/gin"

// SuccessResponse and ErrorResponse are the two JSON envelopes every
// handler in this API responds with — never a bare struct. Keeping
// exactly one shape for success and one for failure means a client
// can always check `success` first and know which shape to expect,
// instead of guessing per-endpoint.
type SuccessResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

// Success writes a SuccessResponse with the given HTTP status.
func Success(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Error writes an ErrorResponse with the given HTTP status. errs may
// be nil — most errors (e.g. "invalid credentials") have nothing
// further to itemize; it's populated mainly for validation failures,
// where each element describes one invalid field.
func Error(c *gin.Context, status int, message string, errs []string) {
	c.JSON(status, ErrorResponse{
		Success: false,
		Message: message,
		Errors:  errs,
	})
}
