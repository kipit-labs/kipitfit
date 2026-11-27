package auth

import (
	"fmt"

	"github.com/kipit-labs/kipitfit/internal/domain"
)

var (
	ErrSignedOut        = domain.Coded(domain.ErrUnauthorized, "signed_out", "sign in first")
	ErrWrongCredentials = domain.Coded(domain.ErrUnauthorized, "invalid_credentials", "wrong email or password")
	ErrWrongSetupToken  = domain.Coded(domain.ErrUnauthorized, "wrong_setup_token", "wrong setup token: it is in the server logs")
	ErrSetupDone        = domain.Coded(domain.ErrInvalid, "setup_done", "setup already completed")
	ErrNotAdmin         = domain.Coded(domain.ErrForbidden, "admin_only", "only admins can do this")
	ErrInvalidEmail     = domain.Coded(domain.ErrInvalid, "invalid_email", "enter a valid email address")
	ErrEmailTaken       = domain.Coded(domain.ErrConflict, "email_taken", "an account already uses this email")
	ErrInvalidInvite    = domain.Coded(domain.ErrInvalid, "invalid_invite", "this invite link is invalid, used or expired")
	ErrInvalidLanguage  = domain.Coded(domain.ErrInvalid, "invalid_language", "language must be en or fr")
	ErrInvalidUnit      = domain.Coded(domain.ErrInvalid, "invalid_unit", "unit must be kg or lb")
	ErrNameTooLong      = domain.Coded(domain.ErrInvalid, "name_too_long", fmt.Sprintf("name must be at most %d characters", maxNameLength))
	errPasswordShort    = domain.Coded(domain.ErrInvalid, "password_too_short", fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	errPasswordLong     = domain.Coded(domain.ErrInvalid, "password_too_long", fmt.Sprintf("password must be at most %d bytes", maxPasswordBytes))
)

func init() {
	ErrNameTooLong.Params = map[string]any{"max": maxNameLength}
	errPasswordShort.Params = map[string]any{"min": minPasswordLength}
	errPasswordLong.Params = map[string]any{"max": maxPasswordBytes}
}
