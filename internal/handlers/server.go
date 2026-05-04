package handlers

import (
	"context"
	"log/slog"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
	"github.com/nikpivkin/roasti-app-backend/internal/articles"
	"github.com/nikpivkin/roasti-app-backend/internal/auth"
	"github.com/nikpivkin/roasti-app-backend/internal/beans"
	"github.com/nikpivkin/roasti-app-backend/internal/follows"
	"github.com/nikpivkin/roasti-app-backend/internal/likes"
	"github.com/nikpivkin/roasti-app-backend/internal/recipes"
	"github.com/nikpivkin/roasti-app-backend/internal/uploads"
	"github.com/nikpivkin/roasti-app-backend/internal/users"
)

// NotificationService handles notification listing and state management.
type NotificationService interface {
	ListNotifications(ctx context.Context, userID string, pag models.PaginationParams) (models.GenericPage[models.Notification], error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkAllRead(ctx context.Context, userID string) error
}

var _ StrictServerInterface = (*ServerHandler)(nil)

// Config holds handler-level configuration.
type Config struct {
	SecureCookies bool
}

// UserLibrary provides access to a user's saved/liked content.
type UserLibrary interface {
	ListLikedRecipes(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedRecipe], error)
	ListLikedArticles(ctx context.Context, currentUserID, targetUserID string, params models.ListUserLikesParams) (models.GenericPage[models.LikedArticle], error)
}

// CommentService handles comment mutations shared across resource types.
type CommentService interface {
	Update(ctx context.Context, userID, commentID, text string) (models.Comment, error)
	Delete(ctx context.Context, userID, commentID string) error
}

// ArticleService handles article creation and feed listing.
type ArticleService interface {
	CreateArticle(ctx context.Context, userID string, req models.CreateArticleRequest) (models.Article, error)
	DeleteArticle(ctx context.Context, userID, articleID string) error
	GetArticle(ctx context.Context, userID, articleID string) (models.Article, error)
	UpdateArticle(ctx context.Context, userID, articleID string, req models.UpdateArticleRequest) (models.Article, error)
	ToggleLike(ctx context.Context, userID, articleID string) (likes.ToggleResult, error)
	ListArticles(ctx context.Context, userID string, params articles.ListArticlesParams) (models.GenericPage[models.Article], error)
	ListComments(ctx context.Context, articleID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error)
	CreateComment(ctx context.Context, userID, articleID, text string, parentID *string) (models.Comment, error)
}

type BeanService interface {
	CreateBean(ctx context.Context, userID string, req models.CreateBeanRequest) (models.Bean, error)
	GetBean(ctx context.Context, userID, beanID string) (models.Bean, error)
	ListBeans(ctx context.Context, userID string, params beans.ListBeansParams) (models.GenericPage[models.Bean], error)
	UpdateBean(ctx context.Context, userID, beanID string, req models.UpdateBeanRequest) (models.Bean, error)
	DeleteBean(ctx context.Context, userID, beanID string) error
	ToggleLike(ctx context.Context, userID, beanID string) (likes.ToggleResult, error)
	CreateComment(ctx context.Context, userID, beanID, text string, parentID *string) (models.Comment, error)
	ListComments(ctx context.Context, beanID string, pag models.PaginationParams) (models.GenericPage[models.CommentThread], error)
}

type ServerHandler struct {
	logger              *slog.Logger
	cfg                 Config
	authService         *auth.Service
	uploadService       *uploads.Service
	userService         *users.Service
	recipeService       *recipes.Service
	articleService      ArticleService
	userLibrary         UserLibrary
	commentService      CommentService
	notificationService NotificationService
	beanService         BeanService
	followService       *follows.Service
}

func NewServerHandler(
	cfg Config,
	recipeService *recipes.Service,
	authService *auth.Service,
	userService *users.Service,
	uploader *uploads.Service,
	articleService ArticleService,
	userLibrary UserLibrary,
	commentService CommentService,
	notificationService NotificationService,
	beanService BeanService,
	followService *follows.Service,
) *ServerHandler {
	return &ServerHandler{
		logger:              slog.Default(),
		cfg:                 cfg,
		recipeService:       recipeService,
		authService:         authService,
		userService:         userService,
		uploadService:       uploader,
		articleService:      articleService,
		userLibrary:         userLibrary,
		commentService:      commentService,
		notificationService: notificationService,
		beanService:         beanService,
		followService:       followService,
	}
}
