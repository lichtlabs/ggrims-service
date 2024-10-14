// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

type AttendeeStatus string

const (
	AttendeeStatusWaiting  AttendeeStatus = "waiting"
	AttendeeStatusAttended AttendeeStatus = "attended"
)

func (e *AttendeeStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = AttendeeStatus(s)
	case string:
		*e = AttendeeStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for AttendeeStatus: %T", src)
	}
	return nil
}

type NullAttendeeStatus struct {
	AttendeeStatus AttendeeStatus
	Valid          bool // Valid is true if AttendeeStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullAttendeeStatus) Scan(value interface{}) error {
	if value == nil {
		ns.AttendeeStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.AttendeeStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullAttendeeStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.AttendeeStatus), nil
}

type TicketStatus string

const (
	TicketStatusAvailable TicketStatus = "available"
	TicketStatusPending   TicketStatus = "pending"
	TicketStatusSold      TicketStatus = "sold"
)

func (e *TicketStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TicketStatus(s)
	case string:
		*e = TicketStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for TicketStatus: %T", src)
	}
	return nil
}

type NullTicketStatus struct {
	TicketStatus TicketStatus
	Valid        bool // Valid is true if TicketStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullTicketStatus) Scan(value interface{}) error {
	if value == nil {
		ns.TicketStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.TicketStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullTicketStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.TicketStatus), nil
}

type Attendee struct {
	ID        pgtype.UUID
	EventID   pgtype.UUID
	TicketID  pgtype.UUID
	Data      []byte
	Status    AttendeeStatus
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

type Event struct {
	ID             pgtype.UUID
	Name           string
	Description    string
	Location       string
	EventStartDate pgtype.Timestamptz
	EventEndDate   pgtype.Timestamptz
	CreatedAt      pgtype.Timestamptz
	UpdatedAt      pgtype.Timestamptz
}

type Payment struct {
	ID         pgtype.UUID
	EventID    pgtype.UUID
	Data       []byte
	Name       string
	Email      string
	BillLinkID int32
	CreatedAt  pgtype.Timestamptz
	UpdatedAt  pgtype.Timestamptz
}

type Ticket struct {
	ID          pgtype.UUID
	EventID     pgtype.UUID
	Name        string
	Description string
	Price       string
	Benefits    []byte
	Status      TicketStatus
	CreatedAt   pgtype.Timestamptz
	UpdatedAt   pgtype.Timestamptz
}

type TicketInput struct {
	ID        pgtype.UUID
	EventID   pgtype.UUID
	Inputs    []byte
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}
