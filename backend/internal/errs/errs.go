package errs

import (
	"errors"
	"fmt"
	"net/http"
)

type Kind int

const (
	KindInternal Kind = iota
	KindBadRequest
	KindUnauthorized
	KindNotFound
	KindUpstream // OM/LLM failed
	KindTimeout
	KindRateLimited
)

type Error struct {
	Kind  Kind
	Msg   string
	Cause error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Cause)
	}
	return e.Msg
}

func (e *Error) Unwrap() error { return e.Cause }

func New(kind Kind, msg string, cause error) *Error {
	return &Error{Kind: kind, Msg: msg, Cause: cause}
}

func BadRequest(msg string) *Error   { return &Error{Kind: KindBadRequest, Msg: msg} }
func Unauthorized(msg string) *Error { return &Error{Kind: KindUnauthorized, Msg: msg} }
func Upstream(msg string, cause error) *Error {
	return &Error{Kind: KindUpstream, Msg: msg, Cause: cause}
}
func Internal(msg string, cause error) *Error {
	return &Error{Kind: KindInternal, Msg: msg, Cause: cause}
}

// HTTPStatus maps an error (including wrapped *Error) to an HTTP status code.
func HTTPStatus(err error) int {
	var e *Error
	if !errors.As(err, &e) {
		return http.StatusInternalServerError
	}
	switch e.Kind {
	case KindBadRequest:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindNotFound:
		return http.StatusNotFound
	case KindUpstream:
		return http.StatusBadGateway
	case KindTimeout:
		return http.StatusGatewayTimeout
	case KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
