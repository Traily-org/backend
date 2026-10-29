// Package response writes HTTP responses using the API envelope:
// successful responses are wrapped in {"data": ...} and errors in
// {"error": {"code": ..., "message": ...}}.
package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	nethttp "net/http"
	"reflect"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/traily-org/server/internal/pkg/apperror"
)

// dataEnvelope is the body of every successful response carrying data.
type dataEnvelope struct {
	Data any `json:"data"`
}

// errorBody describes an error returned to the client.
type errorBody struct {
	Code    apperror.Code `json:"code"`
	Message string        `json:"message"`
}

// errorEnvelope is the body of every error response.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func init() {
	// Report JSON field names (e.g. "created_at") instead of Go struct field
	// names in validation errors, so BindError messages match the payload.
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(jsonFieldName)
	}
}

// JSON writes data wrapped in the {"data": ...} envelope with the given status.
func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, dataEnvelope{Data: data})
}

// NoContent writes an empty 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(nethttp.StatusNoContent)
}

// Error writes err in the error envelope and aborts the request.
//
// When err is (or wraps) an *apperror.Error, its code and message are used and
// the status is derived from apperror.HTTPStatus. Internal application errors
// and any other error are logged and answered with a generic 500
// INTERNAL_ERROR, so internal details never reach the client.
func Error(c *gin.Context, err error) {
	var appErr *apperror.Error
	if errors.As(err, &appErr) && appErr.Code != apperror.CodeInternal {
		WriteError(c, apperror.HTTPStatus(appErr.Code), appErr.Code, appErr.Message)
		return
	}

	log.Error("request failed",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"err", err,
	)
	internal := apperror.Internal(err)
	WriteError(c, nethttp.StatusInternalServerError, internal.Code, internal.Message)
}

// WriteError writes an error envelope with an explicit status, code and
// message, then aborts the request. It is meant for framework-level errors
// that have no apperror counterpart (e.g. 405 Method Not Allowed); handlers
// should use Error instead.
func WriteError(c *gin.Context, status int, code apperror.Code, message string) {
	c.AbortWithStatusJSON(status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

// BindError writes a 400 VALIDATION_ERROR for an error returned by gin's
// binding (ShouldBindJSON, ShouldBindQuery, ...). The message is a readable
// "field: rule" description rather than the raw validator output.
func BindError(c *gin.Context, err error) {
	WriteError(c, nethttp.StatusBadRequest, apperror.CodeValidation, bindErrorMessage(err))
}

// bindErrorMessage converts a binding error into a client-safe message.
func bindErrorMessage(err error) string {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) && len(validationErrs) > 0 {
		msgs := make([]string, 0, len(validationErrs))
		for _, fe := range validationErrs {
			msgs = append(msgs, fe.Field()+": "+ruleMessage(fe))
		}
		return strings.Join(msgs, "; ")
	}

	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.Is(err, io.EOF):
		return "request body is required"
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return "request body is not valid JSON"
	case errors.As(err, &typeErr) && typeErr.Field != "":
		return fmt.Sprintf("%s: must be of type %s", typeErr.Field, jsonTypeName(typeErr.Type))
	default:
		return "invalid request body"
	}
}

// ruleMessage describes the validation rule a field failed.
func ruleMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "uuid", "uuid4":
		return "must be a valid UUID"
	case "url", "http_url":
		return "must be a valid URL"
	case "min":
		if fe.Kind() == reflect.String {
			return "must be at least " + fe.Param() + " characters"
		}
		return "must be at least " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "must be at most " + fe.Param() + " characters"
		}
		return "must be at most " + fe.Param()
	case "len":
		return "must have length " + fe.Param()
	case "gte":
		return "must be greater than or equal to " + fe.Param()
	case "lte":
		return "must be less than or equal to " + fe.Param()
	case "gt":
		return "must be greater than " + fe.Param()
	case "lt":
		return "must be less than " + fe.Param()
	case "oneof":
		return "must be one of: " + strings.Join(strings.Fields(fe.Param()), ", ")
	default:
		return "is invalid"
	}
}

// jsonTypeName returns a JSON-oriented name for a Go type.
func jsonTypeName(t reflect.Type) string {
	if t == nil {
		return "unknown"
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	default:
		return "object"
	}
}

// jsonFieldName returns the JSON name of a struct field, falling back to the
// Go field name when no json tag is set.
func jsonFieldName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	switch name {
	case "-":
		return ""
	case "":
		return f.Name
	default:
		return name
	}
}
