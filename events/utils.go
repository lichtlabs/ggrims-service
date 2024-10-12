package events

import (
	"context"
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

func ChangeTicketsStatus(ctx context.Context, ticketIDs []*string, status string) error {
	if len(ticketIDs) == 0 {
		return nil
	}

	// Create a slice of interfaces to hold ticket IDs for the query arguments
	args := make([]interface{}, len(ticketIDs)+1)
	args[0] = status // The first argument is the new status
	for i, id := range ticketIDs {
		args[i+1] = *id // Add ticket IDs to the args slice
	}

	query := `
		UPDATE tickets
		SET status = $1
		WHERE id IN (` + placeholders(len(ticketIDs)) + `)
	`

	_, err := eventsDb.Exec(ctx, query, args...)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

// Helper function to generate placeholders for SQL query
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	result := "$2"
	for i := 3; i <= n+1; i++ {
		result += fmt.Sprintf(", $%d", i)
	}
	return result
}
