package pagination

type Page[T any] struct {
	Items      []T    `json:"items"`
	Page       uint64 `json:"page"`
	Limit      uint64 `json:"limit"`
	TotalCount int64  `json:"total_count,omitempty"`
}

func NewPage[T any](items []T, p Pagination, totalCount int64) Page[T] {
	return Page[T]{
		Items:      items,
		Page:       p.Page(),
		Limit:      p.Limit(),
		TotalCount: totalCount,
	}
}
