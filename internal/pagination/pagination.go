package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type Pagination struct {
	page   uint64
	limit  uint64
	offset uint64
}

func (p Pagination) Page() uint64   { return p.page }
func (p Pagination) Limit() uint64  { return p.limit }
func (p Pagination) Offset() uint64 { return p.offset }

func New(page, limit int) Pagination {
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return Pagination{
		page:   uint64(page),
		limit:  uint64(limit),
		offset: uint64((page - 1) * limit),
	}
}

func FromRequest(r *http.Request) Pagination {
	q := r.URL.Query()

	var page int
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	var limit int
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	return New(page, limit)
}
