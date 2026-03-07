package output

type ListResult[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type AllResult[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}
