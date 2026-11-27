package domain

import (
	"errors"
	"fmt"
)

// Kinds of failure; the API answers each with its own HTTP status.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalid      = errors.New("invalid input")
)

// Error is a failure the user can act on. Code names it for the web app,
// which shows the message in the user's language (errors.<code> in
// web/src/locales), filled in with Params.
type Error struct {
	Kind   error
	Code   string
	Msg    string
	Params map[string]any
}

func (e *Error) Error() string { return fmt.Sprintf("%v: %s", e.Kind, e.Msg) }
func (e *Error) Unwrap() error { return e.Kind }

// Codes lists every Error code, for the test that each has a translation.
var Codes []string

// Coded declares a user-facing error of kind with its code.
func Coded(kind error, code, msg string) *Error {
	Codes = append(Codes, code)
	return &Error{Kind: kind, Code: code, Msg: msg}
}
