package events

import (
	"context"
	"fmt"
	"math"
	"time"

	"encore.dev/storage/sqldb"
)

// Event represents an event
type Event struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Location       string    `json:"location"`
	EventStartDate time.Time `json:"event_start_date"`
	EventEndDate   time.Time `json:"event_end_date"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// EventWithTicketInputs represents an event with ticket inputs
type EventWithTicketInputs struct {
	Event
	Inputs []*EventTicketInput `json:"inputs"`
}

// BaseResponse represents a base response
type BaseResponse[T any] struct {
	Data    T      `json:"data"`
	Message string `json:"message"`
}

// Metadata represents metadata for a response
type Metadata struct {
	CurrentPage int `json:"current_page"`
	NextPage    int `json:"next_page"`
	TotalPages  int `json:"total_pages"`
	TotalCount  int `json:"total_count"`
	Limit       int `json:"limit"`
}

// ListEventsQuery represents a query to list events
type ListEventsQuery struct {
	Page    int    `query:"page"`
	Limit   int    `query:"limit"`
	Search  string `query:"search"`
	OrderBy string `query:"order_by"`
}

// ListEventsResponse represents a response to list events
type ListEventsResponse struct {
	Events []*Event `json:"events"`
	Meta   Metadata `json:"meta"`
}

// ListEvents lists events
//
//encore:api public method=GET path=/events
func ListEvents(ctx context.Context, query *ListEventsQuery) (*BaseResponse[*ListEventsResponse], error) {
	var events []*Event

	q := ExtractQuery(query)
	sqlQuery := `
		SELECT *
		FROM events
		ORDER BY $1
	`
	var rows *sqldb.Rows
	var err error

	if q.Page == 0 {
		sqlQuery = fmt.Sprintf("%s LIMIT $2", sqlQuery)
		rows, err = eventsDb.Query(ctx, sqlQuery, q.OrderBy, q.Limit)
	} else {
		sqlQuery = fmt.Sprintf("%s LIMIT $2 OFFSET $3", sqlQuery)
		rows, err = eventsDb.Query(ctx, sqlQuery, q.OrderBy, q.Limit, q.Page)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Name, &event.Description, &event.Location, &event.EventStartDate, &event.EventEndDate, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	// select count(*) from events
	var totalCount int
	sqlQueryCount := `
		SELECT count(*) FROM events
		`
	if err := eventsDb.QueryRow(ctx, sqlQueryCount).Scan(&totalCount); err != nil {
		return nil, err
	}

	// if there's no more pages, set next page to 0
	currentPage := q.Page + 1
	nextPage := q.Page + 1
	totalPages := int(math.Ceil(float64(totalCount) / float64(q.Limit)))
	if nextPage > totalPages-1 {
		nextPage = 0
	} else {
		nextPage = nextPage + 1
	}

	return &BaseResponse[*ListEventsResponse]{
		Data: &ListEventsResponse{
			Events: events,
			Meta: Metadata{
				CurrentPage: currentPage,
				NextPage:    nextPage,
				TotalPages:  totalPages,
				TotalCount:  totalCount,
			},
		},
		Message: "Events retrieved successfully",
	}, nil
}

// ListUpcomingEvents lists upcoming events
//
//encore:api public method=GET path=/upcoming-events
func ListUpcomingEvents(ctx context.Context) (*BaseResponse[[]*Event], error) {
	var events []*Event

	sqlQuery := `
		SELECT *
			FROM events
			WHERE NOW() < event_start_date::date
			ORDER BY event_start_date ASC
	`
	var rows *sqldb.Rows
	var err error

	rows, err = eventsDb.Query(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Name, &event.Description, &event.Location, &event.EventStartDate, &event.EventEndDate, &event.CreatedAt, &event.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	// select count(*) from events
	var totalCount int
	sqlQueryCount := `
		SELECT count(*) FROM events
		`
	if err := eventsDb.QueryRow(ctx, sqlQueryCount).Scan(&totalCount); err != nil {
		return nil, err
	}

	return &BaseResponse[[]*Event]{
		Data:    events,
		Message: "Events retrieved successfully",
	}, nil
}

// GetEventRequest represents a request to get an event
type GetEventRequest struct {
	ID string `json:"id"`
}

// GetEvent gets an event
//
//encore:api public method=GET path=/events/:id
func GetEvent(ctx context.Context, id string) (*BaseResponse[*EventWithTicketInputs], error) {
	var event EventWithTicketInputs

	// join event s with event_tickets_inputs
	err := eventsDb.QueryRow(ctx, `
		SELECT
			e.id,
			e.name,
			e.description,
			e.location,
			e.event_start_date,
			e.event_end_date,
			e.created_at,
			e.updated_at,
			eti.inputs
		FROM events e
		LEFT JOIN events_tickets_inputs eti ON e.id = eti.event_id
		WHERE e.id = $1
	`, id).Scan(&event.Event.ID, &event.Event.Name, &event.Event.Description, &event.Event.Location, &event.Event.EventStartDate, &event.Event.EventEndDate, &event.Event.CreatedAt, &event.Event.UpdatedAt, &event.Inputs)
	if err != nil {
		return nil, err
	}

	return &BaseResponse[*EventWithTicketInputs]{
		Data:    &event,
		Message: "Event retrieved successfully",
	}, nil
}

type EventTicketInput struct {
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Type        string  `json:"type"`
	Placeholder *string `json:"placeholder"`
	Required    *bool   `json:"required"`
	Options     []*struct {
		Label string `json:"label"`
		Value string `json:"value"`
	} `json:"options"`
}

// CreateEventRequest represents a request to create an event
type CreateEventRequest struct {
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Location       string              `json:"location"`
	EventStartDate time.Time           `json:"event_start_date"`
	EventEndDate   time.Time           `json:"event_end_date"`
	Inputs         []*EventTicketInput `json:"inputs"`
}

// CreateEventResponse represents a response to create an event
type CreateEventResponse struct {
	Created int `json:"created"`
}

// CreateEvent creates an event
//
//encore:api auth method=POST path=/events
func CreateEvent(ctx context.Context, req *CreateEventRequest) (*BaseResponse[*CreateEventResponse], error) {
	var createdEvent Event

	eventsDb.QueryRow(ctx, `
		INSERT INTO events
			(name, description, location, event_start_date, event_end_date)
		VALUES
			($1, $2, $3, $4, $5)
		RETURNING id
	`, req.Name, req.Description, req.Location, req.EventStartDate, req.EventEndDate).Scan(&createdEvent.ID)

	// insert into event_tickets_inputs with the inputs array as jsonb
	_, err := eventsDb.Exec(ctx, `
			INSERT INTO events_tickets_inputs
				(event_id, inputs)
			VALUES
				($1, $2)
		`, createdEvent.ID, req.Inputs)
	if err != nil {
		return nil, err
	}

	return &BaseResponse[*CreateEventResponse]{
		Data: &CreateEventResponse{
			Created: 1,
		},
		Message: "Event created successfully",
	}, nil
}

// UpdateEventRequest represents a request to update an event
type UpdateEventRequest struct {
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Location       string    `json:"location"`
	EventStartDate time.Time `json:"event_start_date"`
	EventEndDate   time.Time `json:"event_end_date"`
}

// UpdateEventResponse represents a response to update an event
type UpdateEventResponse struct {
	Updated int64 `json:"updated"`
}

// UpdateEvent updates an event
//
//encore:api auth method=PUT path=/events/:id
func UpdateEvent(ctx context.Context, id string, req *UpdateEventRequest) (*BaseResponse[*UpdateEventResponse], error) {
	event := Event{
		Name:           req.Name,
		Description:    req.Description,
		Location:       req.Location,
		EventStartDate: req.EventStartDate,
		EventEndDate:   req.EventEndDate,
	}
	rows, err := eventsDb.Exec(ctx, `
		UPDATE events
		SET
			name = $1,
			description = $2,
			location = $3,
			event_start_date = $4,
			event_end_date = $5
		WHERE id = $6
	`, event.Name, event.Description, event.Location, event.EventStartDate, event.EventEndDate, id)
	if err != nil {
		return nil, err
	}

	return &BaseResponse[*UpdateEventResponse]{
		Data: &UpdateEventResponse{
			Updated: rows.RowsAffected(),
		},
		Message: "Event updated successfully",
	}, nil
}

// DeleteEventRequest represents a request to delete an event
type DeleteEventRequest struct {
	ID string `json:"id"`
}

// DeleteEventResponse represents a response to delete an event
type DeleteEventResponse struct {
	Deleted int64 `json:"deleted"`
}

// DeleteEvent deletes an event
//
//encore:api auth method=DELETE path=/events/:id
func DeleteEvent(ctx context.Context, id string) (*BaseResponse[*DeleteEventResponse], error) {
	rows, err := eventsDb.Exec(ctx, `
		DELETE FROM events
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, err
	}

	return &BaseResponse[*DeleteEventResponse]{
		Data: &DeleteEventResponse{
			Deleted: rows.RowsAffected(),
		},
		Message: "Event deleted successfully",
	}, nil
}

var eventsDb = sqldb.NewDatabase("events", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})
