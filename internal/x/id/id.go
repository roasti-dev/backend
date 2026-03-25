package id

import "github.com/google/uuid"

var NewID = uuid.NewString

func IsValidID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
