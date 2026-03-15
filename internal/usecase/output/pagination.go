package output

type ListResult[T any] struct {
	Items    []T
	Page     int
	PageSize int
	Total    int64
}

type AllResult[T any] struct {
	Items []T
	Total int64
}
