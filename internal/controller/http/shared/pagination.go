package shared

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	_defaultPage     = 1
	_defaultPageSize = 10
	_maxPageSize     = 100
)

type PageMeta struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

type Page[T any] struct {
	List []T `json:"list"`
	PageMeta
}

func NewPage[T any](list []T, page, size int, total int64) Page[T] {
	tp := int((total + int64(size) - 1) / int64(size))
	return Page[T]{List: list, PageMeta: PageMeta{CurrentPage: page, PageSize: size, TotalItems: total, TotalPages: tp}}
}

type PageQuery struct {
	Page     int               `query:"page"`
	PageSize int               `query:"page_size"`
	Keyword  string            `query:"keyword"`
	SortBy   string            `query:"sort_by"`
	Order    string            `query:"order"`
	Filters  map[string]string `query:"-"`
}

type PageQueryOption func(*pageQueryConfig)

type pageQueryConfig struct {
	allowedSortBy  []string
	allowedFilters []string
}

func WithAllowedSortBy(fields ...string) PageQueryOption {
	return func(cfg *pageQueryConfig) {
		cfg.allowedSortBy = fields
	}
}

func WithAllowedFilters(keys ...string) PageQueryOption {
	return func(cfg *pageQueryConfig) {
		cfg.allowedFilters = keys
	}
}

func (pq *PageQuery) Normalize() {
	if pq.Page <= 0 {
		pq.Page = _defaultPage
	}
	if pq.PageSize <= 0 {
		pq.PageSize = _defaultPageSize
	}
	if _maxPageSize > 0 && pq.PageSize > _maxPageSize {
		pq.PageSize = _maxPageSize
	}
	if pq.Filters == nil {
		pq.Filters = map[string]string{}
	}
}

func (pq *PageQuery) AddFilter(key, value string) {
	if pq.Filters == nil {
		pq.Filters = map[string]string{}
	}
	pq.Filters[key] = value
}

func (pq PageQuery) HasFilters() bool {
	return len(pq.Filters) > 0
}

func (pq PageQuery) GetFilters() map[string]string {
	return pq.Filters
}

func (pq PageQuery) Offset() int { return (pq.Page - 1) * pq.PageSize }
func (pq PageQuery) Limit() int  { return pq.PageSize }

func ParsePageQuery(ctx fiber.Ctx) PageQuery { return ParsePageQueryWithOptions(ctx) }

func ParsePageQueryWithOptions(ctx fiber.Ctx, opts ...PageQueryOption) PageQuery {
	cfg := pageQueryConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	page := fiber.Query[int](ctx, "page", _defaultPage)
	pageSize := fiber.Query[int](ctx, "page_size", _defaultPageSize)
	sortBy := ctx.Query("sort_by", "")
	if len(cfg.allowedSortBy) > 0 && sortBy != "" {
		ok := false
		for _, s := range cfg.allowedSortBy {
			if sortBy == s {
				ok = true
				break
			}
		}
		if !ok {
			sortBy = ""
		}
	}
	order := ctx.Query("order", "")
	if sortBy != "" {
		l := strings.ToLower(order)
		if l != "asc" && l != "desc" {
			order = "desc"
		} else {
			order = l
		}
	}
	keyword := ctx.Query("keyword", "")

	pq := PageQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		SortBy:   sortBy,
		Order:    order,
		Filters:  map[string]string{},
	}
	normalize := func(k string) (string, bool) {
		if strings.HasPrefix(k, "filter.") {
			return strings.TrimPrefix(k, "filter."), true
		}
		return "", false
	}
	for k, v := range ctx.Queries() {
		if key, ok := normalize(k); ok && key != "" && v != "" {
			if len(cfg.allowedFilters) > 0 {
				allowed := false
				for _, f := range cfg.allowedFilters {
					if key == f {
						allowed = true
						break
					}
				}
				if !allowed {
					continue
				}
			}
			pq.AddFilter(key, v)
		}
	}
	pq.Normalize()
	return pq
}
