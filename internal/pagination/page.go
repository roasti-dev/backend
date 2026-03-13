package pagination

type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalCount int `json:"total_count,omitempty"`
}

func NewPage[T any](items []T, p Pagination, totalCount int) Page[T] {
	if len(items) == 0 {
		items = []T{}
	}
	return Page[T]{
		Items:      items,
		Page:       p.Page(),
		Limit:      p.Limit(),
		TotalCount: totalCount,
	}
}
