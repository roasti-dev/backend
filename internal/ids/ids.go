package ids

import "github.com/google/uuid"

var NewID = func() string {
	return uuid.NewString()
}
