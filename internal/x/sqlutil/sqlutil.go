package sqlutil

import (
	"database/sql"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func BuildUserPreview(id, username string, name, avatarID sql.NullString) models.UserPreview {
	p := models.UserPreview{Id: id, Username: username}
	if name.Valid {
		p.Name = &name.String
	}
	if avatarID.Valid {
		p.AvatarId = &avatarID.String
	}
	return p
}
