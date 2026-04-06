package handlers

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/x/ptr"
	"github.com/nikpivkin/roasti-app-backend/internal/x/requestctx"
)

func (s *ServerHandler) ListNotifications(ctx context.Context, request ListNotificationsRequestObject) (ListNotificationsResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	pag := models.NewPaginationParams(ptr.FromPtr(request.Params.Page), ptr.FromPtr(request.Params.Limit))
	page, err := s.notificationService.ListNotifications(ctx, userID, pag)
	if err != nil {
		return nil, err
	}
	return ListNotifications200JSONResponse(models.NotificationPage(page)), nil
}

func (s *ServerHandler) GetNotificationUnreadCount(ctx context.Context, request GetNotificationUnreadCountRequestObject) (GetNotificationUnreadCountResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	count, err := s.notificationService.UnreadCount(ctx, userID)
	if err != nil {
		return nil, err
	}
	return GetNotificationUnreadCount200JSONResponse(models.NotificationUnreadCount{
		UnreadCount: int32(count),
	}), nil
}

func (s *ServerHandler) MarkAllNotificationsRead(ctx context.Context, request MarkAllNotificationsReadRequestObject) (MarkAllNotificationsReadResponseObject, error) {
	userID := requestctx.GetUserID(ctx)
	if err := s.notificationService.MarkAllRead(ctx, userID); err != nil {
		return nil, err
	}
	return MarkAllNotificationsRead204Response{}, nil
}
