package ptr

func GetOr[T any](p *T, def T) T {
	if p != nil {
		return *p
	}
	return def
}

func FromPtr[T any](p *T) T {
	if p != nil {
		return *p
	}
	var zero T
	return zero
}

func ToPtr[T any](v T) *T {
	return &v
}
