package handlers

import (
	"context"
)

func (s *ServerHandler) GetHealth(ctx context.Context, request GetHealthRequestObject) (GetHealthResponseObject, error) {
	return GetHealth200TextResponse("OK"), nil
}
