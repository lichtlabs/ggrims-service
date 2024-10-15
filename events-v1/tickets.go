package eventsv1

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"strconv"
	"strings"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/types/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lichtlabs/ggrims-service/events-v1/db"
)

type CreateTicketRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       string   `json:"price"`
	Benefits    []string `json:"benefits"`
	TicketCount int      `json:"ticket_count"`
}

// CreateTickets Create tickets for an event
//
//encore:api auth method=POST path=/v1/events/:id/tickets/create
func CreateTickets(ctx context.Context, id uuid.UUID, req *CreateTicketRequest) (*BaseResponse[InsertionResponse], error) {
	eb := errs.B()

	// Start a database transaction
	tx, err := pgxDB.Begin(ctx)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to start transaction").Err()
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			rlog.Error("An error occurred while rolling back a transaction", "Rollback:err", err.Error())
		}
	}(tx, ctx) // Ensure rollback in case of an error

	benefits, err := json.Marshal(req.Benefits)
	created := 0
	for i := 0; i < req.TicketCount; i++ {
		_, err := query.InsertTicket(ctx, db.InsertTicketParams{
			Name:        req.Name,
			Description: req.Description,
			Price:       req.Price,
			Benefits:    benefits,
			EventID: pgtype.UUID{
				Bytes: id,
				Valid: true,
			},
		})
		if err != nil {
			rlog.Error("An error occurred while creating a ticket", "CreateTicket:err", err.Error())
			return nil, eb.Cause(err).Code(errs.FailedPrecondition).Msg("failed to create tickets").Err()
		}
		created++
	}

	// Commit the transaction if all tickets are created successfully
	if err := tx.Commit(ctx); err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to commit transaction").Err()
	}

	return &BaseResponse[InsertionResponse]{
		Data: InsertionResponse{
			Created: created,
		},
		Message: "Tickets created successfully",
	}, nil
}

type UpdateTicketRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       string   `json:"price"`
	Benefits    []string `json:"benefits"`
	TicketCount int      `json:"ticket_count"`
}

// UpdateTickets Update tickets for an event
//
//encore:api auth method=PUT path=/v1/events/:id/tickets/update
func UpdateTickets(ctx context.Context, id uuid.UUID, req *UpdateTicketRequest) (*BaseResponse[UpdatesResponse], error) {
	eb := errs.B()

	// Start a database transaction
	tx, err := pgxDB.Begin(ctx)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to start transaction").Err()
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			rlog.Error("An error occurred while rolling back a transaction", "Rollback:err", err.Error())
		}
	}(tx, ctx) // Ensure rollback in case of an error

	updated := 0
	for i := 0; i < req.TicketCount; i++ {
		err := query.UpdateTicket(ctx, db.UpdateTicketParams{
			Name:        req.Name,
			Description: req.Description,
			Price:       req.Price,
			Benefits:    []byte(strings.Join(req.Benefits, ",")),
			EventID: pgtype.UUID{
				Bytes: id,
				Valid: true,
			},
		})
		if err != nil {
			rlog.Error("An error occurred while updating a ticket", "UpdateTicket:err", err.Error())
			return nil, eb.Cause(err).Code(errs.FailedPrecondition).Msg("failed to update tickets").Err()
		}
		updated++
	}

	// Commit the transaction if all tickets are updated successfully
	if err := tx.Commit(ctx); err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to commit transaction").Err()
	}

	return &BaseResponse[UpdatesResponse]{
		Data: UpdatesResponse{
			Updated: req.TicketCount,
		},
		Message: "Tickets created successfully",
	}, nil
}

type DeleteTicketRequest struct {
	TicketCount int `json:"ticket_count"`
}

// DeleteTickets Delete tickets on an event
//
//encore:api auth method=DELETE path=/v1/events/:id/tickets/delete
func DeleteTickets(ctx context.Context, id uuid.UUID, req *DeleteTicketRequest) (*BaseResponse[DeletesResponse], error) {
	eb := errs.B()

	// Start a database transaction
	tx, err := pgxDB.Begin(ctx)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to start transaction").Err()
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			rlog.Error("An error occurred while rolling back a transaction", "Rollback:err", err.Error())
		}
	}(tx, ctx) // Ensure rollback in case of an error

	deleted := 0
	for i := 0; i < req.TicketCount; i++ {
		err := query.DeleteTicket(ctx, pgtype.UUID{
			Bytes: id,
			Valid: true,
		})
		if err != nil {
			rlog.Error("An error occurred while deleting a ticket", "DeleteTicket:err", err.Error())
			return nil, eb.Cause(err).Code(errs.FailedPrecondition).Msg("failed to delete tickets").Err()
		}
		deleted++
	}

	// Commit the transaction if all tickets are deleted successfully
	if err := tx.Commit(ctx); err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to commit transaction").Err()
	}

	return &BaseResponse[DeletesResponse]{
		Data: DeletesResponse{
			Deleted: deleted,
		},
		Message: "Tickets deleted successfully",
	}, nil
}

type ListDistinctTicketsResponse struct {
	EventID     pgtype.UUID        `json:"event_id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Price       string             `json:"price"`
	Benefits    []string           `json:"benefits"`
	Status      db.TicketStatus    `json:"status"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
	UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
	Count       int64              `json:"count"`
}

// ListDistinctTickets Get distinct tickets for an event
//
//encore:api public method=GET path=/v1/events/:id/tickets/distinct
func ListDistinctTickets(ctx context.Context, id uuid.UUID) (*BaseResponse[[]ListDistinctTicketsResponse], error) {
	eb := errs.B()

	data, err := query.ListDistinctTicket(ctx, pgtype.UUID{
		Bytes: id,
		Valid: true,
	})
	if err != nil {
		return nil, eb.Code(errs.Internal).Msg("An error occurred while retrieving distinct tickets").Err()
	}

	var tickets []ListDistinctTicketsResponse
	for _, ticket := range data {
		var benefits []string
		err := json.Unmarshal(ticket.Benefits, &benefits)
		if err != nil {
			return nil, eb.Code(errs.Internal).Msg("An error occurred while unmarshalling benefits").Err()
		}

		tickets = append(tickets, ListDistinctTicketsResponse{
			EventID:     ticket.EventID,
			Name:        ticket.Name,
			Description: ticket.Description,
			Price:       ticket.Price,
			Benefits:    benefits,
			Status:      ticket.Status,
			CreatedAt:   ticket.CreatedAt,
			UpdatedAt:   ticket.UpdatedAt,
			Count:       ticket.Count,
		})
	}

	return &BaseResponse[[]ListDistinctTicketsResponse]{
		Data:    tickets,
		Message: "Distinct tickets retrieved successfully",
	}, nil
}

// BuyTicketRequest represents a request to buy a ticket
type BuyTicketRequest struct {
	TicketName   string               `json:"ticket_name"`
	TicketAmount int                  `json:"ticket_amount"`
	Attendees    []*map[string]string `json:"attendees"`
}

// BuyTicketResponse represents a response to buy a ticket
type BuyTicketResponse struct {
	BuyTicketData
	CreateBillResponse
}

// BuyTickets Buy tickets for an event
//
//encore:api public method=POST path=/v1/events/:id/tickets/buy
func BuyTickets(ctx context.Context, id uuid.UUID, req *BuyTicketRequest) (*BaseResponse[BuyTicketResponse], error) {
	eb := errs.B()

	// Start a database transaction
	tx, err := pgxDB.Begin(ctx)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to start transaction").Err()
	}
	defer func(tx pgx.Tx, ctx context.Context) {
		err := tx.Rollback(ctx)
		if err != nil {
			rlog.Error("An error occurred while rolling back a transaction", "Rollback:err", err.Error())
		}
	}(tx, ctx) // Ensure rollback in case of an error

	// get available tickets with name
	availableTickets, err := query.GetAvailableTickets(ctx, db.GetAvailableTicketsParams{
		Name:   req.TicketName,
		Limits: int32(req.TicketAmount),
	})
	if availableTickets == nil {
		return nil, eb.Code(errs.NotFound).Msg("No tickets available").Err()
	}
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Internal).Msg("An error occurred while retrieving available tickets").Err()
	}
	rlog.Info("GetAvailableTickets: ", "availableTickets", availableTickets)

	// call payments
	price, err := strconv.Atoi(availableTickets[0].Price)
	createBillRes, err := CreateBill(ctx, &CreateBillRequest{
		Title:       availableTickets[0].Name,
		Amount:      req.TicketAmount*price + (req.TicketAmount * 1000),
		Type:        "SINGLE",
		ExpiredDate: time.Now().Add(48 * time.Hour),
	})
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Internal).Msg("An error occurred while creating a bill").Err()
	}

	var ticketIds []pgtype.UUID
	for _, ticket := range availableTickets {
		rlog.Info("ChangeTicketsStatus: ", "status", db.TicketStatusPending, "ticketID", ticket.ID)
		err := query.ChangeTicketsStatus(ctx, db.ChangeTicketsStatusParams{
			Status:   db.TicketStatusPending,
			TicketID: ticket.ID,
		})
		if err != nil {
			return nil, eb.Cause(err).Code(errs.Internal).Msg("An error occurred while changing ticket status").Err()
		}

		ticketIds = append(ticketIds, ticket.ID)
	}

	// store buy ticket data
	buyTicketData[fmt.Sprintf("reserve:%d", createBillRes.LinkID)] = BuyTicketData{
		TicketAmount: int(availableTickets[0].Count),
		Attendees:    req.Attendees,
		TicketIDs:    ticketIds,
		EventID: pgtype.UUID{
			Bytes: id,
			Valid: true,
		},
	}

	// Commit the transaction if all tickets are deleted successfully
	if err := tx.Commit(ctx); err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to commit transaction").Err()
	}

	return &BaseResponse[BuyTicketResponse]{
		Data: BuyTicketResponse{
			BuyTicketData{
				EventID:      buyTicketData[fmt.Sprintf("reserve:%d", createBillRes.LinkID)].EventID,
				TicketAmount: req.TicketAmount,
				Attendees:    req.Attendees,
				TicketIDs:    buyTicketData[fmt.Sprintf("reserve:%d", createBillRes.LinkID)].TicketIDs,
			},
			CreateBillResponse{
				LinkID:                createBillRes.LinkID,
				LinkURL:               createBillRes.LinkURL,
				Title:                 createBillRes.Title,
				Type:                  createBillRes.Type,
				Amount:                createBillRes.Amount,
				RedirectURL:           createBillRes.RedirectURL,
				ExpiredDate:           createBillRes.ExpiredDate,
				CreatedFrom:           createBillRes.CreatedFrom,
				Status:                createBillRes.Status,
				Step:                  createBillRes.Step,
				IsAddressRequired:     createBillRes.IsAddressRequired,
				IsPhoneNumberRequired: createBillRes.IsPhoneNumberRequired,
			},
		},
		Message: "Tickets reserved",
	}, nil
}

type BuyTicketData struct {
	EventID      pgtype.UUID          `json:"event_id"`
	TicketAmount int                  `json:"ticket_amount"`
	Attendees    []*map[string]string `json:"attendees"`
	TicketIDs    []pgtype.UUID        `json:"ticket_ids"`
}

// in-memory hashmap to store buy ticket data
var buyTicketData = map[string]BuyTicketData{}
