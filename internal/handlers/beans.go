package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/beans"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) ListBeans(ctx context.Context, request ListBeansRequestObject) (ListBeansResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	page, err := s.beanService.ListBeans(ctx, userID, beans.ListBeansParams{
		Query: request.Params.Q,
		Page:  request.Params.Page,
		Limit: request.Params.Limit,
	})
	if err != nil {
		return nil, err
	}
	return ListBeans200JSONResponse(models.BeanPage{
		Items:      page.Items,
		Pagination: page.Pagination,
	}), nil
}

func (s *ServerHandler) CreateBean(ctx context.Context, request CreateBeanRequestObject) (CreateBeanResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	bean, err := s.beanService.CreateBean(ctx, userID, *request.Body)
	if err != nil {
		return nil, err
	}
	return CreateBean201JSONResponse(bean), nil
}

func (s *ServerHandler) DeleteBean(ctx context.Context, request DeleteBeanRequestObject) (DeleteBeanResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.beanService.DeleteBean(ctx, userID, request.BeanId); err != nil {
		return nil, err
	}
	return DeleteBean204Response{}, nil
}

func (s *ServerHandler) GetBean(ctx context.Context, request GetBeanRequestObject) (GetBeanResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	bean, err := s.beanService.GetBean(ctx, userID, request.BeanId)
	if err != nil {
		return nil, err
	}
	return GetBean200JSONResponse(bean), nil
}

func (s *ServerHandler) ToggleBeanLike(ctx context.Context, request ToggleBeanLikeRequestObject) (ToggleBeanLikeResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	result, err := s.beanService.ToggleLike(ctx, userID, request.BeanId)
	if err != nil {
		return nil, err
	}
	return ToggleBeanLike200JSONResponse(models.ToggleLikeResponse{
		Liked:      result.Liked,
		LikesCount: int32(result.LikesCount),
	}), nil
}

func (s *ServerHandler) UpdateBean(ctx context.Context, request UpdateBeanRequestObject) (UpdateBeanResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	bean, err := s.beanService.UpdateBean(ctx, userID, request.BeanId, *request.Body)
	if err != nil {
		return nil, err
	}
	return UpdateBean200JSONResponse(bean), nil
}
