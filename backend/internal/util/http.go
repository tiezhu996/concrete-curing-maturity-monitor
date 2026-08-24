package util

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const ActorContextKey = "authenticated_actor"

const (
	CodeValidation        = "VALIDATION_ERROR"
	CodeUnauthorized      = "UNAUTHORIZED"
	CodeForbidden         = "FORBIDDEN"
	CodeNotFound          = "NOT_FOUND"
	CodeConflict          = "CONFLICT"
	CodeInvalidTransition = "INVALID_STATE_TRANSITION"
	CodeRateLimited       = "RATE_LIMITED"
	CodeInternal          = "INTERNAL_ERROR"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
	Err     error
}

func (e *AppError) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *AppError) Unwrap() error { return e.Err }

func NewError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func ErrorWithDetails(status int, code, message string, details any) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Details: details}
}

func WrapError(status int, code, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: fmt.Errorf("%s: %w", message, err)}
}

func NotFound(entity string) *AppError {
	return NewError(http.StatusNotFound, CodeNotFound, entity+" was not found")
}

func Conflict(message string) *AppError {
	return NewError(http.StatusConflict, CodeConflict, message)
}

type Actor struct {
	UserID      uint   `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	RequestID   string `json:"request_id"`
}

func ActorFromGin(c *gin.Context) (Actor, bool) {
	value, ok := c.Get(ActorContextKey)
	if !ok {
		return Actor{}, false
	}
	actor, ok := value.(Actor)
	if !ok {
		return Actor{}, false
	}
	actor.RequestID = RequestID(c)
	return actor, true
}

func RequestID(c *gin.Context) string {
	if value, ok := c.Get("request_id"); ok {
		if requestID, cast := value.(string); cast {
			return requestID
		}
	}
	return c.GetHeader("X-Request-ID")
}

type Envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"request_id"`
}

type ErrorEnvelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   any    `json:"details,omitempty"`
	RequestID string `json:"request_id"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: "OK", Message: "success", Data: data, RequestID: RequestID(c)})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Code: "OK", Message: "created", Data: data, RequestID: RequestID(c)})
}

func WriteError(c *gin.Context, err error) {
	var appError *AppError
	if !errors.As(err, &appError) {
		appError = WrapError(http.StatusInternalServerError, CodeInternal, "the request could not be completed", err)
	}
	c.AbortWithStatusJSON(appError.Status, ErrorEnvelope{
		Code: appError.Code, Message: appError.Message, Details: appError.Details, RequestID: RequestID(c),
	})
}

func BindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		WriteError(c, ErrorWithDetails(http.StatusBadRequest, CodeValidation, "request body is invalid", err.Error()))
		return false
	}
	return true
}

func ParseID(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		WriteError(c, NewError(http.StatusBadRequest, CodeValidation, name+" must be a positive integer"))
		return 0, false
	}
	return uint(value), true
}

func Pagination(c *gin.Context) (int, int) {
	page := parseBoundedInt(c.Query("page"), 1, 1, 100000)
	size := parseBoundedInt(c.Query("page_size"), 20, 1, 100)
	return page, size
}

func parseBoundedInt(value string, fallback, minimum, maximum int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < minimum || parsed > maximum {
		return fallback
	}
	return parsed
}
