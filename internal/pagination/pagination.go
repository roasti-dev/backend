package pagination

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type Pagination struct {
	page  int
	limit int
}

func (p Pagination) Page() int   { return p.page }
func (p Pagination) Limit() int  { return p.limit }
func (p Pagination) Offset() int { return (p.page - 1) * p.limit }

func New(page, limit int) Pagination {
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return Pagination{
		page:  page,
		limit: limit,
	}
}
