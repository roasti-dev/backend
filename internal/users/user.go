package users

import (
	"time"

	"github.com/oapi-codegen/nullable"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

type User struct {
	ID        string
	Email     string
	Username  string
	Name      *string
	AvatarID  *string
	Bio       *string
	CreatedAt time.Time
}

func (u User) ToPreview() models.UserPreview {
	return models.UserPreview{
		Id:       u.ID,
		Username: u.Username,
		Name:     u.Name,
		AvatarId: u.AvatarID,
	}
}

func (u User) ToPublicProfile() models.UserProfile {
	return models.UserProfile{
		Id:       u.ID,
		Username: u.Username,
		Name:     u.Name,
		AvatarId: u.AvatarID,
		Bio:      u.Bio,
	}
}

func (u User) ToAccount() models.UserAccount {
	return models.UserAccount{
		Id:       u.ID,
		Email:    u.Email,
		Username: u.Username,
		Name:     u.Name,
		AvatarId: u.AvatarID,
		Bio:      u.Bio,
	}
}

// UpdateUserFields holds the fields to update. nil / zero value means "not provided, skip".
type UpdateUserFields struct {
	Username *string
	Name     nullable.Nullable[string]
	Bio      nullable.Nullable[string]
	AvatarID nullable.Nullable[string]
}

func (r UpdateUserFields) HasFields() bool {
	return r.Username != nil || r.Name.IsSpecified() || r.Bio.IsSpecified() || r.AvatarID.IsSpecified()
}
