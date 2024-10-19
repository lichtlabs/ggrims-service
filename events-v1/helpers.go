package eventsv1

import (
	"fmt"
	"strings"
)

// extractQuery adjusts the given ListQuery based on predefined rules.
// It modifies the page to be zero-based, sets a default limit if not provided,
// and formats the order by clause with default or specified values.
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
