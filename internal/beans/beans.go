package beans

// ListBeansParams holds parameters for listing beans from the catalog.
type ListBeansParams struct {
	Query *string
	Page  *int32
	Limit *int32
}
