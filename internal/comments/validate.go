package comments

import "strings"

func normalizeComment(text string) string {
	return strings.TrimSpace(text)
}

func validateComment(text string) error {
	if text == "" {
		return ErrInvalidText
	}
	if len(text) > textMaxLen {
		return ErrTextTooLong
	}
	return nil
}
