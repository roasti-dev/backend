package posts

import "github.com/nikpivkin/roasti-app-backend/internal/api/models"

const (
	postTitleMaxLen    = 200
	blocksMaxCount     = 30
	blockTextMaxLen    = 5000
	commentTextMaxLen  = 1000
)

func validatePostPayload(req models.PostPayload) error {
	if req.Title == "" {
		return ErrInvalidTitle
	}
	if len(req.Title) > postTitleMaxLen {
		return ErrTitleTooLong
	}
	if len(req.Blocks) > blocksMaxCount {
		return ErrTooManyBlocks
	}
	for _, block := range req.Blocks {
		if block.Text != nil && len(*block.Text) > blockTextMaxLen {
			return ErrBlockTextTooLong
		}
	}
	return nil
}

func validateCommentText(text string) error {
	if text == "" {
		return ErrInvalidCommentText
	}
	if len(text) > commentTextMaxLen {
		return ErrCommentTextTooLong
	}
	return nil
}
