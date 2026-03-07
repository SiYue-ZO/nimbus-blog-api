package input

import (
	"strconv"
	"strings"
)

type PageParams struct {
	Page     int
	PageSize int
}

type KeywordParams struct {
	Keyword string
}

type SortParams struct {
	SortBy string
	Order  string
}

type StringFilterParam *string

type IntFilterParam *int

type BoolFilterParam *bool

// ParseStringFilterParam 解析字符串过滤参数（空字符串返回 nil）。
func ParseStringFilterParam(s string) StringFilterParam {
	if s == "" {
		return nil
	}
	v := s
	return StringFilterParam(&v)
}

// ParseIntFilterParam 解析整数过滤参数（空字符串/解析失败返回 nil）。
func ParseIntFilterParam(s string) IntFilterParam {
	if s == "" {
		return nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	v := i
	return IntFilterParam(&v)
}

// ParseBoolFilterParam 解析布尔过滤参数（空字符串/非法值返回 nil）。
func ParseBoolFilterParam(s string) BoolFilterParam {
	if s == "" {
		return nil
	}
	switch strings.ToLower(s) {
	case "true":
		b := true
		return BoolFilterParam(&b)
	case "false":
		b := false
		return BoolFilterParam(&b)
	default:
		return nil
	}
}

// StringFilterParamToString 转换 StringFilterParam（nil 返回空字符串）。
func StringFilterParamToString(p StringFilterParam) string {
	if p == nil {
		return ""
	}
	return *p
}

// IntFilterParamToInt 转换 IntFilterParam（nil 返回 0）。
func IntFilterParamToInt(p IntFilterParam) int {
	if p == nil {
		return 0
	}
	return *p
}

// BoolFilterParamToBool 转换 BoolFilterParam（nil 返回 false）。
func BoolFilterParamToBool(p BoolFilterParam) bool {
	if p == nil {
		return false
	}
	return *p
}
