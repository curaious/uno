package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	json "github.com/bytedance/sonic"
	"github.com/valyala/fasthttp"

	"github.com/curaious/uno/internal/perrors"
)

type Response[T any] struct {
	ctx          context.Context
	ErrorDetails perrors.Err `json:"errorDetails"`
	Error        bool        `json:"error"`
	Message      string      `json:"message"`
	Data         T           `json:"data"`
	Status       int         `json:"status"`
}

func NewResponse[T any](ctx context.Context, msg string, data T) *Response[T] {
	return &Response[T]{
		ctx:     ctx,
		Message: msg,
		Data:    data,
		Status:  http.StatusOK,
	}
}

// WithError sets the error field for the response
func (r *Response[T]) WithError(err error) *Response[T] {
	// Set http status from error if available
	var perr perrors.Err
	if errors.As(err, &perr) {
		r.Status = perr.HttpStatus()
		r.ErrorDetails = perr
		perr.Print(r.ctx)
	} else {
		perr = perrors.NewErrInternalServerError(r.Message, err).(perrors.Err)
		r.ErrorDetails = perr
		perr.Print(r.ctx)
	}

	r.Error = true

	return r
}

// WithStatus will set the HTTP response status code.
//
// This is not a preferred way of setting status code.
//   - Try to use perrors.Err embedded with a status code whenever possible.
//   - Default is http.StatusOK and it need not be set explicitly.
func (r *Response[T]) WithStatus(code int) *Response[T] {
	r.Status = code

	return r
}

// Write will set the `content-type` to `application/json` and write the response to the fasthttp context.
func (r *Response[T]) Write(ctx *fasthttp.RequestCtx) {
	if r.Error {
		slog.ErrorContext(r.ctx, "Error processing the request", slog.Any("error", r.ErrorDetails))
	}

	ctx.Response.Header.Set("content-type", "application/json")
	ctx.SetStatusCode(r.Status)

	body, err := json.Marshal(r)
	if err != nil {
		slog.ErrorContext(r.ctx, "Unable to json encode response", slog.Any("error", err))
		ctx.SetStatusCode(http.StatusInternalServerError)
		return
	}

	ctx.SetBody(body)
}
