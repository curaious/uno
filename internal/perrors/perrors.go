package perrors

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
)

type ErrCode struct {
	Code   string `json:"code"`
	Status int    `json:"status"`
}

var (
	ErrCodeInvalidRequest      ErrCode = ErrCode{"invalid_request", http.StatusBadRequest}
	ErrCodeInternalServer              = ErrCode{"internal_server_error", http.StatusInternalServerError}
	ErrCodeNotFound                    = ErrCode{"not_found", http.StatusNotFound}
	ErrCodeConflict                    = ErrCode{"conflict", http.StatusConflict}
	ErrCodeUnauthorized                = ErrCode{"unauthorized", http.StatusUnauthorized}
	ErrCodeForbidden                   = ErrCode{"forbidden", http.StatusForbidden}
	ErrCodeBadRequest                  = ErrCode{"bad_request", http.StatusBadRequest}
	ErrCodeMethodNotAllowed            = ErrCode{"method_not_allowed", http.StatusMethodNotAllowed}
	ErrCodeTooManyRequests             = ErrCode{"too_many_requests", http.StatusTooManyRequests}
	ErrCodeInternalServerError         = ErrCode{"internal_server_error", http.StatusInternalServerError}
	ErrCodeNotImplemented              = ErrCode{"not_implemented", http.StatusNotImplemented}
)

type Err struct {
	Message    string                   `json:"-"`
	Err        string                   `json:"error"`
	Code       ErrCode                  `json:"-"`
	Stacktrace []string                 `json:"-"`
	Args       []map[string]interface{} `json:"args"`
}

func (e Err) Error() string {
	return e.Err
}

func (e Err) HttpStatus() int {
	return e.Code.Status
}

func (e Err) Print(ctx context.Context) {
	args := append([]any{slog.Any("error", e.Error())})
	if len(e.Args) > 0 {
		for k, v := range e.Args[0] {
			args = append(args, slog.Any(k, v))
		}
	}
	args = append(args, slog.Any("stacktrace", e.Stacktrace))
	slog.ErrorContext(ctx, e.Message, args...)
}

func New(code ErrCode, msg string, err error, args ...map[string]interface{}) error {
	pc := make([]uintptr, 20)
	count := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:count])

	var stacktrace []string
	for frame, hasMore := frames.Next(); hasMore; frame, hasMore = frames.Next() {
		stacktrace = append(stacktrace, fmt.Sprintf("%s:%d", frame.File, frame.Line))
	}

	errString := "error missing"
	if err != nil {
		errString = err.Error()
	}

	return Err{
		Code:       code,
		Message:    msg,
		Err:        errString,
		Stacktrace: stacktrace,
		Args:       args,
	}
}

func NewErrInvalidRequest(msg string, err error, args ...map[string]interface{}) error {
	return New(ErrCodeInvalidRequest, msg, err, args...)
}

func NewErrInternalServerError(msg string, err error, args ...map[string]interface{}) error {
	return New(ErrCodeInternalServer, msg, err, args...)
}
