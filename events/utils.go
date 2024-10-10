package events

import (
	"fmt"
	"strings"
)

func ExtractQuery(query *ListEventsQuery) *ListEventsQuery {
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
	if query.Search == "" {
		query.Search = "*"
	}
	return query
}
