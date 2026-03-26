package users

import (
	"regexp"
)

const (
	PasswordMinLength = 8
	PasswordMaxLength = 32
)

var (
	passwordUpperRegex = regexp.MustCompile(`[A-Z]`)
	passwordDigitRegex = regexp.MustCompile(`[0-9]`)
)

type Password struct {
	raw string
}

func NewPassword(raw string) (Password, error) {
	if err := validatePassword(raw); err != nil {
		return Password{}, err
	}
	return Password{raw: raw}, nil
}

func validatePassword(raw string) error {
	if len(raw) < PasswordMinLength {
		return ErrPasswordTooShort
	}

	if len(raw) > PasswordMaxLength {
		return ErrPasswordTooLong
	}

	// if !passwordUpperRegex.MatchString(raw) || !passwordDigitRegex.MatchString(raw) {
	// 	return ErrInvalidPasswordFormat
	// }
	return nil
}

func (p Password) Value() string {
	return p.raw
}

func (p Password) String() string {
	return "[PROTECTED]"
}
