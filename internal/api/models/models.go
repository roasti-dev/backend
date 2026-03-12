package models

import (
	"github.com/nikpivkin/roasti-app-backend/internal/pagination"
	"github.com/nikpivkin/roasti-app-backend/internal/ptr"
)

func (p ListRecipesParams) Pagination() pagination.Pagination {
	return pagination.New(ptr.FromPtr(p.Page), ptr.FromPtr(p.Limit))
}
