package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

func (s *ServerHandler) PostApiV1UploadsImages(ctx context.Context, request PostApiV1UploadsImagesRequestObject) (PostApiV1UploadsImagesResponseObject, error) {
	id, err := s.uploadsService.UploadMultipart(ctx, request.Body)
	if err != nil {
		return nil, err
	}
	return PostApiV1UploadsImages201JSONResponse(models.Image{Id: id}), nil
}

func (s *ServerHandler) GetApiV1UploadsImagesImageId(ctx context.Context, request GetApiV1UploadsImagesImageIdRequestObject) (GetApiV1UploadsImagesImageIdResponseObject, error) {
	img, err := s.uploadsService.Resolve(ctx, request.ImageId)
	if err != nil {
		return nil, err
	}
	// defer img.Body.Close()
	return GetApiV1UploadsImagesImageId200ImageResponse{
		Body:          img.Body,
		ContentType:   img.ContentType,
		ContentLength: img.Size,
	}, nil
}
