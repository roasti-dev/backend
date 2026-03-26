// internal/users/username.go
package users

import (
	"regexp"
)

const (
	usernameMinLength = 6
	usernameMaxLength = 16
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type Username struct {
	value string
}

func NewUsername(raw string) (Username, error) {
	if err := validateUsername(raw); err != nil {
		return Username{}, err
	}
	return Username{value: raw}, nil
}

func (u Username) Value() string {
	return u.value
}

func (u Username) String() string {
	return u.value
}

func validateUsername(raw string) error {
	if len(raw) < usernameMinLength {
		return ErrUsernameTooShort
	}

	if len(raw) > usernameMaxLength {
		return ErrUsernameTooLong
	}

	if !usernameRegex.MatchString(raw) {
		return ErrInvalidUsernameFormat
	}
	return nil
}
