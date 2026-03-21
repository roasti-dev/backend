package likes

import (
	"time"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

type Like struct {
	ID         string
	UserID     string
	TargetID   string
	TargetType models.LikeTargetType
	CreatedAt  time.Time
}
