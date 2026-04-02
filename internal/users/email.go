package users

import (
	"net/mail"
	"strings"
)

type Email struct {
	value string
}

func NewEmail(raw string) (Email, error) {
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return Email{}, ErrInvalidEmailFormat
	}
	return Email{value: strings.ToLower(addr.Address)}, nil
}

func (e Email) Value() string {
	return e.value
}

func (e Email) String() string {
	return e.value
}
