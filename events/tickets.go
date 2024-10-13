package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"
)

// Ticket represents a event ticket
type Ticket struct {
	ID          string    `json:"id"`
	EventID     string    `json:"event_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	Benefits    []string  `json:"benefits"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Attendee map[string]string

// CreateTicketRequest represents a request to create a ticket for admin
type CreateTicketRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       string   `json:"price"`
	Benefits    []string `json:"benefits"`
	// Amount of tickets to create
	TicketCount int `json:"ticket_count"`
}

// CreateTicketResponse represents a response to create a ticket for admin
type CreateTicketResponse struct {
	Created int `json:"created"`
}

// CreateTicket creates tickets for admin in bulk
//
//encore:api auth method=POST path=/events/:id/tickets/create
func CreateTicket(ctx context.Context, id string, req *CreateTicketRequest) (*BaseResponse[*CreateTicketResponse], error) {
	// Start a transaction
	tx, err := eventsDb.Begin(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure transaction is rolled back in case of a failure
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for i := 0; i < req.TicketCount; i++ {
		_, err := tx.Exec(ctx, `
			INSERT INTO tickets
				(event_id, name, description, price, benefits)
			VALUES
				($1, $2, $3, $4, $5)
		`, id, req.Name, req.Description, req.Price, req.Benefits)

		if err != nil {
			// An error occurred, rollback the transaction
			return nil, err
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &BaseResponse[*CreateTicketResponse]{
		Data: &CreateTicketResponse{
			Created: req.TicketCount,
		},
		Message: "Tickets created successfully",
	}, nil
}

// ReserveTicketResponse represents a response to buy a ticket
type ReserveTicketResponse struct {
	TicketIDs []string `json:"ticket_ids"`
}

// ReserveTicket reserve a ticket handler for payments service to hit
// when the payment status is SUCCESSFUL
//
//encore:api private method=POST path=/tickets/reserve/:linkID
func ReserveTicket(ctx context.Context, linkID int) (*BaseResponse[*ReserveTicketResponse], error) {
	var err error
	// Get buy ticket data from memory
	buyTicketsData := buyTicketData[fmt.Sprintf("reserve:%d", linkID)]
	log.Println("Buy tickets data (reserve):", buyTicketsData)

	tx, err := eventsDb.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	query := `
        UPDATE tickets
        SET status = 'sold'
        WHERE id = ANY($1)
    `
	_, err = tx.Exec(ctx, query, buyTicketsData.TicketIDs)
	if err != nil {
		return nil, err
	}

	// drop buy ticket data
	delete(buyTicketData, fmt.Sprintf("reserve:%d", linkID))

	return &BaseResponse[*ReserveTicketResponse]{
		Data:    &ReserveTicketResponse{TicketIDs: buyTicketsData.TicketIDs},
		Message: "Tickets reserved",
	}, nil
}

// RollbackTicketsResponse represents a response to buy a ticket
type RollbackTicketsResponse struct {
	TicketIDs []string `json:"ticket_ids"`
}

// RollbackTickets rollback a ticket status to available
//
//encore:api private method=POST path=/tickets/rollback/:linkID
func RollbackTickets(ctx context.Context, linkID int) (*BaseResponse[*RollbackTicketsResponse], error) {
	var err error
	// Get buy ticket data from memory
	buyTicketsData := buyTicketData[fmt.Sprintf("reserve:%d", linkID)]
	log.Println("Buy tickets data (rollback):", buyTicketsData)

	tx, err := eventsDb.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	query := `
        UPDATE tickets
        SET status = 'available'
        WHERE id = ANY($1)
    `
	_, err = tx.Exec(ctx, query, buyTicketsData.TicketIDs)
	if err != nil {
		return nil, err
	}

	return &BaseResponse[*RollbackTicketsResponse]{
		Data:    &RollbackTicketsResponse{TicketIDs: buyTicketsData.TicketIDs},
		Message: "Tickets rollbacked",
	}, nil
}

// ListTicketsResponse represents a response to list tickets
type ListTicketsResponse struct {
	EventID     string    `json:"event_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       string    `json:"price"`
	Benefits    []string  `json:"benefits"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Count       int       `json:"count"`
}

// ListTickets get distinct tickets by its name
//
//encore:api public method=GET path=/events/:id/tickets
func ListTickets(ctx context.Context, id string) (*BaseResponse[[]*ListTicketsResponse], error) {
	var tickets []*ListTicketsResponse

	rows, err := eventsDb.Query(ctx, `
		SELECT DISTINCT ON (name)
			event_id,
			name,
			description,
			price,
			benefits,
			status,
			created_at,
			updated_at,
				COUNT(*)
				OVER (PARTITION BY name)
			AS count

		FROM tickets
		WHERE event_id = $1 AND status = 'available'
		ORDER BY name, created_at DESC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ticket ListTicketsResponse

		if err := rows.Scan(&ticket.EventID, &ticket.Name, &ticket.Description, &ticket.Price, &ticket.Benefits, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt, &ticket.Count); err != nil {
			return nil, err
		}

		tickets = append(tickets, &ListTicketsResponse{
			EventID:     ticket.EventID,
			Name:        ticket.Name,
			Description: ticket.Description,
			Price:       ticket.Price,
			Benefits:    ticket.Benefits,
			Status:      ticket.Status,
			CreatedAt:   ticket.CreatedAt,
			UpdatedAt:   ticket.UpdatedAt,
			Count:       ticket.Count,
		})
	}

	return &BaseResponse[[]*ListTicketsResponse]{
		Data:    tickets,
		Message: "Tickets retrieved successfully",
	}, nil

}

// BuyTicketRequest represents a request to buy a ticket
type BuyTicketRequest struct {
	TicketName   string      `json:"ticket_name"`
	TicketAmount int         `json:"ticket_amount"`
	Attendees    []*Attendee `json:"attendees"`
}

// BuyTicketResponse represents a response to buy a ticket
type BuyTicketResponse struct {
	BuyTicketData
	CreateBillResponse
}

// BuyTicket buys a ticket
//
//encore:api public method=POST path=/events/:id/tickets/buy
func BuyTicket(ctx context.Context, id string, req *BuyTicketRequest) (*BaseResponse[*BuyTicketResponse], error) {
	tx, txErr := eventsDb.Begin(ctx)
	if txErr != nil {
		return nil, txErr
	}
	defer func() {
		if p := recover(); p != nil {
			log.Println("Error: panic: ", p)
			tx.Rollback()
			panic(p)
		}
		if txErr != nil {
			log.Println("Error: txErr: ", txErr)
			tx.Rollback()
		} else {
			log.Println("commit")
			tx.Commit()
		}
	}()

	var ticketCount int

	// check tickets count before buying
	res := tx.QueryRow(ctx, `
		SELECT count(*) FROM tickets
		WHERE event_id = $1
		AND status = 'available'
		AND name = $2
	`, id, req.TicketName)

	err := res.Scan(&ticketCount)
	if err != nil {
		txErr = err
		return nil, err
	}

	log.Println("ticketCount: ", ticketCount)

	if ticketCount < req.TicketAmount {
		return nil, errors.New("Not enough tickets available")
	}

	var tickets []*Ticket
	ticketIDs := make([]string, 0, req.TicketAmount)

	rows, err := tx.Query(ctx, `
		SELECT id, event_id, name, description, price, benefits, status, created_at, updated_at
		FROM tickets
		WHERE event_id = $1
		AND status = 'available'
		AND name = $3
		LIMIT $2
		FOR UPDATE
	`, id, req.TicketAmount, req.TicketName)

	if err != nil {
		txErr = err
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ticket Ticket
		if err := rows.Scan(&ticket.ID, &ticket.EventID, &ticket.Name, &ticket.Description, &ticket.Price, &ticket.Benefits, &ticket.Status, &ticket.CreatedAt, &ticket.UpdatedAt); err != nil {
			txErr = err
			return nil, err
		}
		tickets = append(tickets, &ticket)
		ticketIDs = append(ticketIDs, ticket.ID)
	}

	if len(tickets) < req.TicketAmount {
		return &BaseResponse[*BuyTicketResponse]{
			Data:    nil,
			Message: "Not enough tickets available",
		}, nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE tickets
		SET status = 'pending'
		WHERE id = ANY($1)
	`, ticketIDs)
	if err != nil {
		txErr = err
		return nil, err
	}

	totalPrice := 0
	for _, ticket := range tickets {
		price, err := strconv.Atoi(ticket.Price)
		if err != nil {
			txErr = err
			return nil, err
		}
		totalPrice += price + 1000 // add 1000 for the payment fee
	}

	createBillReq, err := CreateBill(ctx, &CreateBillRequest{
		Title:                 tickets[0].Name,
		Amount:                totalPrice,
		Type:                  "SINGLE",
		ExpiredDate:           time.Now().Add(48 * time.Hour),
		IsAddressRequired:     0,
		IsPhoneNumberRequired: 0,
	})
	if err != nil {
		txErr = err
		return nil, err
	}

	buyTicketData[fmt.Sprintf("reserve:%d", createBillReq.LinkID)] = BuyTicketData{
		TicketAmount: req.TicketAmount,
		Attendees:    req.Attendees,
		TicketIDs:    ticketIDs,
	}

	log.Println("Buy tickets data (buy):", buyTicketData)

	return &BaseResponse[*BuyTicketResponse]{
		Data: &BuyTicketResponse{
			BuyTicketData{
				TicketAmount: req.TicketAmount,
				Attendees:    req.Attendees,
				TicketIDs:    ticketIDs,
			},
			CreateBillResponse{
				LinkID:                createBillReq.LinkID,
				LinkURL:               createBillReq.LinkURL,
				Title:                 createBillReq.Title,
				Type:                  createBillReq.Type,
				Amount:                createBillReq.Amount,
				RedirectURL:           createBillReq.RedirectURL,
				ExpiredDate:           createBillReq.ExpiredDate,
				CreatedFrom:           createBillReq.CreatedFrom,
				Status:                createBillReq.Status,
				Step:                  createBillReq.Step,
				IsAddressRequired:     createBillReq.IsAddressRequired,
				IsPhoneNumberRequired: createBillReq.IsPhoneNumberRequired,
			},
		},
		Message: "Tickets reserved",
	}, nil
}

type BuyTicketData struct {
	TicketAmount int         `json:"ticket_amount"`
	Attendees    []*Attendee `json:"attendees"`
	TicketIDs    []string    `json:"ticket_ids"`
}

// in-memory hashmap to store buy ticket data
var buyTicketData = map[string]BuyTicketData{}
