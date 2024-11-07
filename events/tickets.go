package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"encore.dev/types/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lichtlabs/ggrims-service/events/db"
)

// LetterBytes is a constant string containing alphanumeric characters used for generating random strings.
const LetterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// CreateTicketRequest represents a request to create a new ticket for an event. It includes the ticket's name,
// description, price, associated benefits, and the total number of tickets to create.
type CreateTicketRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       string   `json:"price"`
	Benefits    []string `json:"benefits"`
	TicketCount int      `json:"ticket_count"`
	Min         int      `json:"min"`
	Max         int      `json:"max"`
}

// CreateTickets creates multiple tickets for an event and inserts them into the database within a transaction.
// It takes a context, event ID (UUID), and a CreateTicketRequest object as input.
// Returns a BaseResponse containing the count of created tickets and a success message, or an error if operation fails.
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
			rlog.Error("An error occurred while rolling back the transaction", "Rollback:err", err.Error())
		}
	}(tx, ctx)

	ticketHash := func(n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = LetterBytes[rand.Intn(len(LetterBytes))]
		}
		return string(b)
	}

	benefits, err := json.Marshal(req.Benefits)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to marshal err").Err()
	}

	created := 0
	for i := 0; i < req.TicketCount; i++ {
		_, err := query.InsertTicket(ctx, db.InsertTicketParams{
			Name:        req.Name,
			Description: req.Description,
			Price:       req.Price,
			Benefits:    benefits,
			Hash: pgtype.Text{
				String: ticketHash(32),
				Valid:  true,
			},
			EventID: pgtype.UUID{
				Bytes: id,
				Valid: true,
			},
			Min: pgtype.Int4{
				Int32: int32(req.Min),
				Valid: true,
			},
			Max: pgtype.Int4{
				Int32: int32(req.Max),
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

// UpdateTicketRequest represents a request structure for updating ticket information.
type UpdateTicketRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       string   `json:"price"`
	Benefits    []string `json:"benefits"`
	TicketCount int      `json:"ticket_count"`
}

// UpdateTickets updates a specified number of tickets in the database within a single transaction.
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
			rlog.Error("An error occurred while rolling back the transaction", "Rollback:err", err.Error())
		}
	}(tx, ctx)

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
		log.Println("failed to commit transaction", err)
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to commit transaction").Err()
	}

	return &BaseResponse[UpdatesResponse]{
		Data: UpdatesResponse{
			Updated: req.TicketCount,
		},
		Message: "Tickets created successfully",
	}, nil
}

// DeleteTicketRequest represents a request to delete a specified number of tickets by name.
type DeleteTicketRequest struct {
	TicketName  string `json:"ticket_name"`
	TicketCount int    `json:"ticket_count"`
}

// DeleteTickets removes a specified number of tickets based on the given `DeleteTicketRequest`.
// It returns a `BaseResponse` containing `DeletesResponse` with the number of deleted tickets or an error.
//
//encore:api auth method=DELETE path=/v1/events/:id/tickets/delete
func DeleteTickets(ctx context.Context, id uuid.UUID, req *DeleteTicketRequest) (*BaseResponse[DeletesResponse], error) {
	eb := errs.B()

	err := query.DeleteTicket(ctx, db.DeleteTicketParams{
		TicketName: req.TicketName,
		Limits:     int32(req.TicketCount),
		EvID: pgtype.UUID{
			Bytes: id,
			Valid: true,
		},
	})
	if err != nil {
		rlog.Error("An error occurred while deleting a ticket", "DeleteTicket:err", err.Error())
		return nil, eb.Cause(err).Code(errs.FailedPrecondition).Msg("failed to delete tickets").Err()
	}

	return &BaseResponse[DeletesResponse]{
		Data: DeletesResponse{
			Deleted: req.TicketCount,
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
	Min         int32              `json:"min"`
	Max         int32              `json:"max"`
	CreatedAt   pgtype.Timestamptz `json:"created_at"`
	UpdatedAt   pgtype.Timestamptz `json:"updated_at"`
	Count       int64              `json:"count"`
}

// ListDistinctTickets retrieves a list of distinct tickets based on a given event ID.
// It returns a BaseResponse containing a list of ListDistinctTicketsResponse or an error if the operation fails.
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
			Min:         ticket.Min.Int32,
			Max:         ticket.Max.Int32,
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

// BuyTicketRequest represents the payload required to purchase tickets for an event.
type BuyTicketRequest struct {
	TicketName   string               `json:"ticket_name"`
	TicketAmount int                  `json:"ticket_amount"`
	ReferralCode string               `json:"referral_code"`
	Attendees    []*map[string]string `json:"attendees"`
}

// BuyTicketResponse combines the data of a ticket purchase and the billing response.
type BuyTicketResponse struct {
	BuyTicketData
	CreateBillResponse
}

// BuyTickets processes the ticket purchase request, handles the database transaction, and manages billing for the tickets.
//
//encore:api public method=POST path=/v1/events/:id/tickets/buy
func BuyTickets(ctx context.Context, id uuid.UUID, req *BuyTicketRequest) (*BaseResponse[BuyTicketResponse], error) {
	eb := errs.B()

	// Start a database transaction
	tx, err := pgxDB.Begin(ctx)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Unavailable).Msg("failed to start transaction").Err()
	}

	var committed bool
	defer func() {
		if !committed {
			err := tx.Rollback(ctx)
			if err != nil && err != pgx.ErrTxClosed {
				rlog.Error("failed to rollback transaction", "err", err.Error())
			}
		}
	}()

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

	price, err := strconv.Atoi(availableTickets[0].Price)
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Internal).Msg("failed to convert price to int").Err()
	}

	// Calculate price with referral code if provided
	var discountAmount int
	if req.ReferralCode != "" {
		refCode, err := validateReferralCode(ctx, req.ReferralCode)
		if err != nil {
			return nil, eb.Cause(err).Code(errs.InvalidArgument).Msg("invalid referral code").Err()
		}

		if refCode != nil {
			totalAmount := req.TicketAmount * price
			discountAmount = (totalAmount * int(refCode.DiscountPercentage)) / 100
			price = totalAmount - discountAmount
		}
	}

	// Create bill with discounted price if applicable
	jakartaLoc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Internal).Msg("failed to load Jakarta timezone").Err()
	}

	createBillRes, err := CreateBill(ctx, &CreateBillRequest{
		Title:                 availableTickets[0].Name,
		Amount:                price + (req.TicketAmount * 1000), // Adding fixed fee
		Type:                  "SINGLE",
		ExpiredDate:           time.Now().In(jakartaLoc).Add(7 * time.Minute),
		IsAddressRequired:     0,
		IsPhoneNumberRequired: 0,
	})
	if err != nil {
		return nil, eb.Cause(err).Code(errs.Internal).Msg("An error occurred while creating a bill").Err()
	}
	rlog.Info("CreateBillResponse: ", "createBillRes", createBillRes)

	var ticketIds []pgtype.UUID
	var ticketHashes []string
	for _, ticket := range availableTickets {
		if err := query.ChangeTicketsStatus(ctx, db.ChangeTicketsStatusParams{
			Status:   db.TicketStatusPending,
			TicketID: ticket.ID,
		}); err != nil {
			return nil, eb.Cause(err).Code(errs.Internal).Msg("An error occurred while changing ticket status").Err()
		}

		ticketIds = append(ticketIds, ticket.ID)
		ticketHashes = append(ticketHashes, ticket.Hash.String)
	}

	// store buy ticket data
	reserveKey := fmt.Sprintf("reserve:%d", createBillRes.LinkID)
	buyTicketData[reserveKey] = BuyTicketData{
		TicketAmount: int(availableTickets[0].Count),
		Attendees:    req.Attendees,
		TicketIDs:    ticketIds,
		TicketHashes: ticketHashes,
		EventID: pgtype.UUID{
			Bytes: id,
			Valid: true,
		},
		ReferralCode: req.ReferralCode,
	}

	// Start a goroutine to handle the timeout
	go func() {
		rlog.Info("Checking payment existence", "billLinkID", createBillRes.LinkID)
		time.Sleep(7 * time.Minute)

		// Check if payment exists
		paymentExists, err := query.CheckPaymentExists(context.Background(), int32(createBillRes.LinkID))
		if err != nil {
			rlog.Error("Error checking payment existence", "err", err)
			return
		}

		if !paymentExists {
			// No payment received, change ticket status back to available
			for _, ticketID := range ticketIds {
				err := query.ChangeTicketsStatus(context.Background(), db.ChangeTicketsStatusParams{
					Status:   db.TicketStatusAvailable,
					TicketID: ticketID,
				})
				if err != nil {
					rlog.Error("Error reverting ticket status", "ticketID", ticketID, "err", err)
				}
			}
			// Remove the reservation data
			delete(buyTicketData, reserveKey)
			rlog.Info("Reverted ticket statuses due to no payment", "billLinkID", createBillRes.LinkID)
		} else {
			rlog.Info("Payment received", "billLinkID", createBillRes.LinkID)
		}
	}()

	// Commit the transaction if all tickets are deleted successfully
	if err := tx.Commit(ctx); err != nil {
		rlog.Error("failed to commit your transaction", "err", err.Error())
		return nil, eb.Cause(err).Code(errs.DataLoss).Msg("failed to commit your transaction").Err()
	}
	committed = true

	return &BaseResponse[BuyTicketResponse]{
		Data: BuyTicketResponse{
			BuyTicketData{
				EventID:      buyTicketData[reserveKey].EventID,
				TicketAmount: req.TicketAmount,
				Attendees:    req.Attendees,
				TicketIDs:    buyTicketData[reserveKey].TicketIDs,
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

// BuyTicketData holds the data required for purchasing tickets for an event.
type BuyTicketData struct {
	EventID      pgtype.UUID          `json:"event_id"`
	TicketAmount int                  `json:"ticket_amount"`
	Attendees    []*map[string]string `json:"attendees"`
	TicketIDs    []pgtype.UUID        `json:"ticket_ids"`
	TicketHashes []string             `json:"ticket_hashes"`
	ReferralCode string               `json:"referral_code"`
}

// buyTicketData is a map that stores temporary BuyTicketData keyed by a unique payment link_id identifier.
var buyTicketData = map[string]BuyTicketData{}
