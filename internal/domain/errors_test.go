package domain

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorConstructorsAndMatching(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code ErrorCode
	}{
		{"invalid", InvalidArgumentError("", errors.New("x")), ErrorCodeInvalidArgument},
		{"not_found", NotFoundError("", errors.New("x")), ErrorCodeNotFound},
		{"conflict", ConflictError("", errors.New("x")), ErrorCodeConflict},
		{"unauthorized", UnauthorizedError("", errors.New("x")), ErrorCodeUnauthorized},
		{"forbidden", ForbiddenError("", errors.New("x")), ErrorCodeForbidden},
		{"service", ServiceError("", errors.New("x")), ErrorCodeService},
		{"internal", InternalError(errors.New("x")), ErrorCodeInternal},
	}

	for _, tc := range cases {
		if !IsErrorCode(tc.err, tc.code) {
			t.Fatalf("%s: expected code %s", tc.name, tc.code)
		}
	}
}

func TestToGRPCStatusAlwaysInternal(t *testing.T) {
	if ToGRPCStatus(nil) != nil {
		t.Fatalf("nil error must map to nil grpc status")
	}
	st := status.Convert(ToGRPCStatus(InvalidArgumentError("bad", errors.New("x"))))
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal grpc code, got %s", st.Code())
	}
	if st.Message() != "internal server error" {
		t.Fatalf("unexpected message: %q", st.Message())
	}
}
