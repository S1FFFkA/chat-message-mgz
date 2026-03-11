package domain

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrorCode string

const (
	ErrorCodeInvalidArgument ErrorCode = "invalid_argument"
	ErrorCodeNotFound        ErrorCode = "not_found"
	ErrorCodeConflict        ErrorCode = "conflict"
	ErrorCodeUnauthorized    ErrorCode = "unauthorized"
	ErrorCodeForbidden       ErrorCode = "forbidden"
	ErrorCodeService         ErrorCode = "service_unavailable"
	ErrorCodeInternal        ErrorCode = "internal"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func InvalidArgumentError(message string, err error) *AppError {
	if message == "" {
		message = "invalid request"
	}
	return NewError(ErrorCodeInvalidArgument, message, err)
}

func NotFoundError(message string, err error) *AppError {
	if message == "" {
		message = "resource not found"
	}
	return NewError(ErrorCodeNotFound, message, err)
}

func ConflictError(message string, err error) *AppError {
	if message == "" {
		message = "resource conflict"
	}
	return NewError(ErrorCodeConflict, message, err)
}

func UnauthorizedError(message string, err error) *AppError {
	if message == "" {
		message = "unauthorized"
	}
	return NewError(ErrorCodeUnauthorized, message, err)
}

func ForbiddenError(message string, err error) *AppError {
	if message == "" {
		message = "forbidden"
	}
	return NewError(ErrorCodeForbidden, message, err)
}

func ServiceError(message string, err error) *AppError {
	if message == "" {
		message = "service unavailable"
	}
	return NewError(ErrorCodeService, message, err)
}

func InternalError(err error) *AppError {
	return NewError(ErrorCodeInternal, "internal server error", err)
}

func IsErrorCode(err error, code ErrorCode) bool {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code == code
}

// ToGRPCStatus intentionally always returns Internal for client safety.
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(codes.Internal, "internal server error")
}
