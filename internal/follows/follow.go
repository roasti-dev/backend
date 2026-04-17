package follows

import "time"

const TargetTypeUser = "user"

type Follow struct {
	ID         string
	FollowerID string
	TargetID   string
	TargetType string
	CreatedAt  time.Time
}
