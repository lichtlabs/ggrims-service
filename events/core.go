package events

import "github.com/jackc/pgx/v5/pgtype"

type Event struct {
	ID             pgtype.UUID         `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Location       string              `json:"location"`
	EventStartDate pgtype.Timestamptz  `json:"event_start_date"`
	EventEndDate   pgtype.Timestamptz  `json:"event_end_date"`
	Disabled       bool                `json:"disabled"`
	CreatedAt      pgtype.Timestamptz  `json:"created_at"`
	UpdatedAt      pgtype.Timestamptz  `json:"updated_at"`
	TicketInputs   []*EventTicketInput `json:"inputs"`
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

type Ticket struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Price        string             `json:"price"`
	Benefits     []byte             `json:"benefits"`
	Status       string             `json:"status"`
	TicketInputs []byte             `json:"ticket_inputs"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
	UpdatedAt    pgtype.Timestamptz `json:"updated_at"`
}

type Attendee struct {
	ID        string             `json:"id"`
	EventID   string             `json:"event_id"`
	TicketID  string             `json:"ticket_id"`
	Data      []byte             `json:"data"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

type Payment struct {
	ID         string             `json:"id"`
	EventID    string             `json:"event_id"`
	TicketID   string             `json:"ticket_id"`
	Data       []byte             `json:"data"`
	Name       string             `json:"name"`
	Email      string             `json:"email"`
	BillLinkID int                `json:"bill_link_id"`
	CreatedAt  pgtype.Timestamptz `json:"created_at"`
	UpdatedAt  pgtype.Timestamptz `json:"updated_at"`
}
