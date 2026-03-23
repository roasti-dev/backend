package models

import (
	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
)

func (p ListRecipesParams) Pagination() PaginationParams {
	return NewPaginationParams(ptr.FromPtr(p.Page), ptr.FromPtr(p.Limit))
}

func (p ListMyLikesParams) Pagination() PaginationParams {
	return NewPaginationParams(ptr.FromPtr(p.Page), ptr.FromPtr(p.Limit))
}
