package users

import (
	"time"

	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

type User struct {
	ID        string
	Email     string
	Username  string
	AvatarID  *string
	Bio       *string
	CreatedAt time.Time
}

func (u User) ToPublicProfile() models.UserProfile {
	return models.UserProfile{
		Id:       u.ID,
		Username: u.Username,
		AvatarId: u.AvatarID,
		Bio:      u.Bio,
	}
}

func (u User) ToAccount() models.UserAccount {
	return models.UserAccount{
		Id:       u.ID,
		Email:    openapi_types.Email(u.Email),
		Username: u.Username,
		AvatarId: u.AvatarID,
		Bio:      u.Bio,
	}
}

// UpdateUserFields holds the fields to update. nil / zero value means "not provided, skip".
type UpdateUserFields struct {
	Username *string
	Bio      nullable.Nullable[string]
	AvatarID nullable.Nullable[string]
}

func (r UpdateUserFields) HasFields() bool {
	return r.Username != nil || r.Bio.IsSpecified() || r.AvatarID.IsSpecified()
}
