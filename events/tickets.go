package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"encore.app/payments"
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

// ReserveTicketRequest represents a request to buy a ticket
type ReserveTicketRequest struct {
	LinkID int `json:"link_id"`
}

// ReserveTicketResponse represents a response to buy a ticket
type ReserveTicketResponse struct {
	TicketIDs []*string `json:"ticket_ids"`
}

// ReserveTicket reserve a ticket handler for payments service to hit
// when the payment status is SUCCESSFUL
//
//encore:api private method=POST path=/events/:id/tickets/reserve
func ReserveTicket(ctx context.Context, id string, req *ReserveTicketRequest) (*BaseResponse[*ReserveTicketResponse], error) {
	var err error
	// Get buy ticket data from memory
	buyTicketsData := buyTicketData[fmt.Sprintf("reserve:%d", req.LinkID)]

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

	return nil, nil
}

// BuyTicketRequest represents a request to buy a ticket
type BuyTicketRequest struct {
	TicketAmount int         `json:"ticket_amount"`
	Attendees    []*Attendee `json:"attendees"`
}

// BuyTicketResponse represents a response to buy a ticket
type BuyTicketResponse struct {
	BuyTicketData
	payments.CreateBillResponse
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
			log.Println("panic: ", p)
			tx.Rollback()
			panic(p)
		}
		if txErr != nil {
			log.Println("txErr: ", txErr)
			tx.Rollback()
		} else {
			log.Println("commit")
			tx.Commit()
		}
	}()

	// check tickets count before buying
	res := tx.QueryRow(ctx, `
		SELECT count(*) FROM tickets
		WHERE event_id = $1
		AND status = 'available'
	`, id)
	var ticketCount int
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
	ticketIDs := make([]*string, 0, req.TicketAmount)

	rows, err := tx.Query(ctx, `
		SELECT id, event_id, name, description, price, benefits, status, created_at, updated_at
		FROM tickets
		WHERE event_id = $1
		AND status = 'available'
		LIMIT $2
		FOR UPDATE
	`, id, req.TicketAmount)

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
		ticketIDs = append(ticketIDs, &ticket.ID)
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
		totalPrice += price
	}

	log.Printf("Exp Date: %s", time.Now().Add(15*time.Minute).Format("2006-01-02"))

	createBillReq, err := payments.CreateBill(ctx, &payments.CreateBillRequest{
		Title:                 "Ticket purchase",
		Amount:                totalPrice,
		Type:                  "SINGLE",
		ExpiredDate:           time.Now().Add(48 * time.Hour),
		RedirectURL:           "",
		IsAddressRequired:     0,
		IsPhoneNumberRequired: 0,
	})
	if err != nil {
		txErr = err
		return nil, err
	}

	buyTicketData[fmt.Sprintf("reserve:%d", createBillReq.LinkID)] = &BuyTicketData{
		TicketAmount: req.TicketAmount,
		Attendees:    req.Attendees,
		TicketIDs:    ticketIDs,
	}

	return &BaseResponse[*BuyTicketResponse]{
		Data: &BuyTicketResponse{
			BuyTicketData{
				TicketAmount: req.TicketAmount,
				Attendees:    req.Attendees,
				TicketIDs:    ticketIDs,
			},
			payments.CreateBillResponse{
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
	TicketIDs    []*string   `json:"ticket_ids"`
}

// in-memory hashmap to store buy ticket data
var buyTicketData = map[string]*BuyTicketData{}
