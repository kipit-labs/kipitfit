package core

import (
	"errors"
	"fmt"
)

// Kinds of failure; the API answers each with its own HTTP status.
var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalid      = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
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

func coded(kind error, code, msg string) *Error {
	Codes = append(Codes, code)
	return &Error{Kind: kind, Code: code, Msg: msg}
}

var (
	ErrSignedOut        = coded(ErrUnauthorized, "signed_out", "sign in first")
	ErrWrongCredentials = coded(ErrUnauthorized, "invalid_credentials", "wrong email or password")
	ErrWrongSetupToken  = coded(ErrUnauthorized, "wrong_setup_token", "wrong setup token: it is in the server logs")
	ErrSetupDone        = coded(ErrInvalid, "setup_done", "setup already completed")
	ErrNotAdmin         = coded(ErrForbidden, "admin_only", "only admins can do this")
	ErrInvalidEmail     = coded(ErrInvalid, "invalid_email", "enter a valid email address")
	ErrEmailTaken       = coded(ErrConflict, "email_taken", "an account already uses this email")
	ErrInvalidInvite    = coded(ErrInvalid, "invalid_invite", "this invite link is invalid, used or expired")
	ErrInvalidLanguage  = coded(ErrInvalid, "invalid_language", "language must be en or fr")
	ErrInvalidUnit      = coded(ErrInvalid, "invalid_unit", "unit must be kg or lb")
	ErrNameTooLong      = coded(ErrInvalid, "name_too_long", fmt.Sprintf("name must be at most %d characters", maxNameLength))
	errPasswordShort    = coded(ErrInvalid, "password_too_short", fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	errPasswordLong     = coded(ErrInvalid, "password_too_long", fmt.Sprintf("password must be at most %d bytes", maxPasswordBytes))
)

func init() {
	ErrNameTooLong.Params = map[string]any{"max": maxNameLength}
	errPasswordShort.Params = map[string]any{"min": minPasswordLength}
	errPasswordLong.Params = map[string]any{"max": maxPasswordBytes}
}
