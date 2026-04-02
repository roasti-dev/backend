package posts

import (
	"fmt"
	"net/http"

	"github.com/nikpivkin/roasti-app-backend/internal/api/apierr"
)

var (
	ErrNotFound           = apierr.NewApiError(http.StatusNotFound, "post not found")
	ErrCommentNotFound    = apierr.NewApiError(http.StatusNotFound, "comment not found")
	ErrForbidden          = apierr.NewApiError(http.StatusForbidden, "not allowed")
	ErrInvalidTitle       = apierr.NewApiError(http.StatusUnprocessableEntity, "title cannot be empty")
	ErrTitleTooLong       = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("title must be at most %d characters", postTitleMaxLen))
	ErrTooManyBlocks      = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("post must have at most %d blocks", blocksMaxCount))
	ErrBlockTextTooLong   = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("block text must be at most %d characters", blockTextMaxLen))
	ErrInvalidCommentText = apierr.NewApiError(http.StatusUnprocessableEntity, "comment text cannot be empty")
	ErrCommentTextTooLong = apierr.NewApiError(http.StatusUnprocessableEntity, fmt.Sprintf("comment text must be at most %d characters", commentTextMaxLen))
)
