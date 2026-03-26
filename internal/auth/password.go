package auth

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	passwordUpperRegex = regexp.MustCompile(`[A-Z]`)
	passwordDigitRegex = regexp.MustCompile(`[0-9]`)
)

// Password is a validated password value object.
type Password struct {
	raw string
}

// NewPassword validates raw against policy and returns a Password on success.
func NewPassword(raw string, policy PasswordPolicy) (Password, error) {
	if len(raw) < policy.MinLength {
		return Password{}, apierr.NewApiError(http.StatusUnprocessableEntity,
			fmt.Sprintf("password must be at least %d characters", policy.MinLength))
	}
	if len(raw) > policy.MaxLength {
		return Password{}, apierr.NewApiError(http.StatusUnprocessableEntity,
			fmt.Sprintf("password must be at most %d characters", policy.MaxLength))
	}
	if policy.RequireUppercase && !passwordUpperRegex.MatchString(raw) {
		return Password{}, apierr.NewApiError(http.StatusUnprocessableEntity,
			"password must contain at least one uppercase letter")
	}
	if policy.RequireNumeric && !passwordDigitRegex.MatchString(raw) {
		return Password{}, apierr.NewApiError(http.StatusUnprocessableEntity,
			"password must contain at least one digit")
	}
	return Password{raw: raw}, nil
}

func (p Password) Value() string {
	return p.raw
}

func (p Password) String() string {
	return "[PROTECTED]"
}
