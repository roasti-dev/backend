package users

import "time"

type User struct {
	ID        string
	Email     string
	Username  string
	AvatarID  *string
	Bio       *string
	CreatedAt time.Time
}

// UpdateUserFields holds the fields to update. nil means "not provided, skip".
type UpdateUserFields struct {
	Username *string
	Bio      *string
	AvatarID *string
}

func (r UpdateUserFields) HasFields() bool {
	return r.Username != nil || r.Bio != nil || r.AvatarID != nil
}
