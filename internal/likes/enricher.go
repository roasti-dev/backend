package likes

import (
	"context"

	"github.com/nikpivkin/roasti-app-backend/internal/api/models"
)

type Likeable interface {
	LikeTargetID() string
	LikeTargetType() models.LikeTargetType
	ApplyLikeInfo(isLiked bool, count int)
}

type Checker interface {
	GetInfo(ctx context.Context, userID, targetID string, targetType models.LikeTargetType) (Info, error)
	GetInfoBatch(ctx context.Context, userID string, targetType models.LikeTargetType, targetIDs []string) (map[string]Info, error)
}

type Enricher struct {
	checker Checker
}

func NewEnricher(checker Checker) *Enricher {
	return &Enricher{checker: checker}
}

func (e *Enricher) EnrichOne(ctx context.Context, userID string, item Likeable) error {
	info, err := e.checker.GetInfo(ctx, userID, item.LikeTargetID(), item.LikeTargetType())
	if err != nil {
		return err
	}
	item.ApplyLikeInfo(info.IsLiked, info.Count)
	return nil
}

func (e *Enricher) EnrichMany(ctx context.Context, userID string, items []Likeable) error {
	if len(items) == 0 {
		return nil
	}

	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.LikeTargetID()
	}

	targetType := items[0].LikeTargetType()
	infos, err := e.checker.GetInfoBatch(ctx, userID, targetType, ids)
	if err != nil {
		return err
	}

	for i := range items {
		if info, ok := infos[items[i].LikeTargetID()]; ok {
			items[i].ApplyLikeInfo(info.IsLiked, info.Count)
		}
	}
	return nil
}
