package models

import (
	"log/slog"
	"math"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

func (p PaginationParams) GetLimit() int32 {
	if p.Limit == nil {
		return DefaultLimit
	}
	return *p.Limit
}

func (p PaginationParams) GetPage() int32 {
	if p.Page == nil {
		return DefaultPage
	}
	return *p.Page
}

func (p PaginationParams) Offset() int32 { return (p.GetPage() - 1) * p.GetLimit() }

func NewPaginationParams(page, limit int32) PaginationParams {
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return PaginationParams{
		Limit: new(limit),
		Page:  new(page),
	}
}

func (p PaginationMeta) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("items_count", int(p.ItemsCount)),
		slog.Int("current_page", int(p.CurrentPage)),
		slog.Int("next_page", int(p.NextPage)),
		slog.Int("last_page", int(p.LastPage)),
	)
}

type GenericPage[T any] struct {
	Items      []T            `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

func EmptyPage[T any]() GenericPage[T] {
	return NewPage([]T{}, NewPaginationParams(0, 0), 0)
}

func NewPage[T any](items []T, pag PaginationParams, total int) GenericPage[T] {
	if len(items) == 0 {
		items = []T{}
	}

	lastPage := max(1, int32(math.Ceil(float64(total)/float64(pag.GetLimit()))))
	currentPage := pag.GetPage()
	nextPage := min(currentPage+1, lastPage)

	return GenericPage[T]{
		Items: items,
		Pagination: PaginationMeta{
			ItemsCount:  int32(len(items)),
			CurrentPage: currentPage,
			NextPage:    nextPage,
			LastPage:    lastPage,
		},
	}
}
