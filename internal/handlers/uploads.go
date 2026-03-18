package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func (s *ServerHandler) UploadImage(ctx context.Context, request UploadImageRequestObject) (UploadImageResponseObject, error) {
	id, err := s.uploadsService.UploadMultipart(ctx, request.Body)
	if err != nil {
		return nil, err
	}
	return UploadImage201JSONResponse(models.Image{Id: id}), nil
}

func (s *ServerHandler) GetImage(ctx context.Context, request GetImageRequestObject) (GetImageResponseObject, error) {
	img, err := s.uploadsService.Resolve(ctx, request.ImageId)
	if err != nil {
		return nil, err
	}
	return GetImage200ImageResponse{
		Body:          img.Body,
		ContentType:   img.ContentType,
		ContentLength: img.Size,
	}, nil
}
