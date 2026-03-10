package pagination

type PaginatedResult[T any] struct {
	Items      []T    `json:"items"`
	Page       uint64 `json:"page"`
	Limit      uint64 `json:"limit"`
	TotalCount int64  `json:"total_count,omitempty"`
}

func NewResult[T any](items []T, p Pagination, totalCount int64) PaginatedResult[T] {
	return PaginatedResult[T]{
		Items:      items,
		Page:       p.Page(),
		Limit:      p.Limit(),
		TotalCount: totalCount,
	}
}
