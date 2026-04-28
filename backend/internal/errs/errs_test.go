package errs

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestHTTPStatus_KindMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{BadRequest("x"), http.StatusBadRequest},
		{Unauthorized("x"), http.StatusUnauthorized},
		{Upstream("x", nil), http.StatusBadGateway},
		{Internal("x", nil), http.StatusInternalServerError},
		{New(KindTimeout, "x", nil), http.StatusGatewayTimeout},
		{New(KindRateLimited, "x", nil), http.StatusTooManyRequests},
		{New(KindNotFound, "x", nil), http.StatusNotFound},
		{errors.New("plain"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		if got := HTTPStatus(tc.err); got != tc.want {
			t.Errorf("HTTPStatus(%v) = %d, want %d", tc.err, got, tc.want)
		}
	}
}

func TestError_WrapsCause(t *testing.T) {
	cause := errors.New("root")
	e := Internal("wrapped", cause)
	if !errors.Is(e, cause) {
		t.Fatal("errors.Is should unwrap to cause")
	}
	if e.Error() == "wrapped" {
		t.Fatalf("expected cause to be in message: got %q", e.Error())
	}
}

func TestError_AsStruct(t *testing.T) {
	e := BadRequest("oops")
	wrapped := fmt.Errorf("ctx: %w", e)
	var got *Error
	if !errors.As(wrapped, &got) {
		t.Fatal("expected errors.As to find *Error")
	}
	if got.Kind != KindBadRequest {
		t.Fatalf("kind mismatch: %v", got.Kind)
	}
}
