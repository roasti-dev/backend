package articles

import "github.com/nikpivkin/roasti-app-backend/internal/api/models"

const (
	articleTitleMaxLen = 200
	blocksMaxCount     = 30
	blockTextMaxLen    = 5000
)

func validateArticlePayload(req models.ArticlePayload) error {
	if req.Title == "" {
		return ErrInvalidTitle
	}
	if len(req.Title) > articleTitleMaxLen {
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
