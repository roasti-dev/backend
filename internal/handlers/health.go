package handlers

import (
	"context"
)

func (s *ServerHandler) HealthCheck(ctx context.Context, request HealthCheckRequestObject) (HealthCheckResponseObject, error) {
	return HealthCheck200TextResponse("OK"), nil
}
