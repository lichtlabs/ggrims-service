package eventsv1

import (
	"fmt"
	"strings"
)

func baseResponse[T any](data T, message string) *BaseResponse[T] {
	return &BaseResponse[T]{
		Data:    data,
		Message: message,
	}
}

// ExtractQuery extracts query from query string
func extractQuery(query *ListQuery) *ListQuery {
	if query.Page > 0 {
		query.Page = query.Page - 1
	}
	if query.Limit == 0 {
		query.Limit = 10
	}
	if query.OrderBy == "" {
		query.OrderBy = "created_at DESC"
	} else {
		query.OrderBy = fmt.Sprintf("%s %s", strings.Split(query.OrderBy, ":")[0], strings.Split(query.OrderBy, ":")[1])
	}

	return query
}
