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
