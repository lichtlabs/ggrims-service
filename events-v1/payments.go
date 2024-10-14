package eventsv1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"github.com/lichtlabs/ggrims-service/events-v1/db"
)

type CreateBillRequest struct {
	Title                 string    `json:"title"`
	Amount                int       `json:"amount"`
	Type                  string    `json:"type"`
	ExpiredDate           time.Time `json:"expired_date"`
	RedirectURL           string    `json:"redirect_url"`
	IsAddressRequired     int       `json:"is_address_required"`
	IsPhoneNumberRequired int       `json:"is_phone_number_required"`
}

type CreateBillResponse struct {
	LinkID                int    `json:"link_id"`
	LinkURL               string `json:"link_url"`
	Title                 string `json:"title"`
	Type                  string `json:"type"`
	Amount                int    `json:"amount"`
	RedirectURL           string `json:"redirect_url"`
	ExpiredDate           string `json:"expired_date"`
	CreatedFrom           string `json:"created_from"`
	Status                string `json:"status"`
	Step                  int    `json:"step"`
	IsAddressRequired     int    `json:"is_address_required"`
	IsPhoneNumberRequired int    `json:"is_phone_number_required"`
}

// CreateBill creates a bill for users to pay
//
//encore:api private method=POST path=/payments
func CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	eb := errs.B()

	createBillEndpoint := fmt.Sprintf("%s/pwf/bill", secrets.FlipApiBaseEndpoint)
	data := url.Values{}
	data.Set("title", req.Title)
	data.Set("amount", fmt.Sprintf("%d", req.Amount))
	data.Set("type", req.Type)
	data.Set("expired_date", req.ExpiredDate.Format("2006-01-02"))
	data.Set("redirect_url", req.RedirectURL)
	data.Set("is_address_required", fmt.Sprintf("%d", req.IsAddressRequired))
	data.Set("is_phone_number_required", fmt.Sprintf("%d", req.IsPhoneNumberRequired))

	encodedCredentials := base64.StdEncoding.EncodeToString([]byte(secrets.FlipApiSecretKey + ":"))
	reqs, err := http.NewRequest(http.MethodPost, createBillEndpoint, bytes.NewBufferString(data.Encode()))
	if err != nil {
		rlog.Info("Error creating request:", "err", err)
		return nil, eb.Code(errs.Internal).Msg("Error creating request").Err()
	}

	reqs.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqs.Header.Set("Authorization", "Basic "+encodedCredentials)

	client := &http.Client{}
	resp, err := client.Do(reqs)
	if err != nil {
		rlog.Info("Error making request:", "err", err)
		return nil, eb.Code(errs.Internal).Msg("Error making request").Err()
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			rlog.Error("Error closing response body", "err", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		rlog.Info("Error reading response:", "err", err)
		return nil, eb.Code(errs.Internal).Msg("Error reading response").Err()
	}
	rlog.Info("Response:", string(body))

	var jsonResponse CreateBillResponse
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, err
	}

	return &CreateBillResponse{
		LinkID:                jsonResponse.LinkID,
		LinkURL:               jsonResponse.LinkURL,
		Title:                 jsonResponse.Title,
		Type:                  jsonResponse.Type,
		Amount:                jsonResponse.Amount,
		RedirectURL:           jsonResponse.RedirectURL,
		ExpiredDate:           jsonResponse.ExpiredDate,
		CreatedFrom:           jsonResponse.CreatedFrom,
		Status:                jsonResponse.Status,
		Step:                  jsonResponse.Step,
		IsAddressRequired:     jsonResponse.IsAddressRequired,
		IsPhoneNumberRequired: jsonResponse.IsPhoneNumberRequired,
	}, nil
}

type Transaction struct {
	ID             string `json:"id"`
	BillLink       string `json:"bill_link"`
	BillLinkID     int    `json:"bill_link_id"`
	BillTitle      string `json:"bill_title"`
	SenderName     string `json:"sender_name"`
	SenderBank     string `json:"sender_bank"`
	SenderEmail    string `json:"sender_email"`
	Amount         int    `json:"amount"`
	Status         string `json:"status"`
	SenderBankType string `json:"sender_bank_type"`
	CreatedAt      string `json:"created_at"`
}

type CallbackRequest struct {
	Data  Transaction `json:"data"`
	Token string      `json:"token,omitempty"`
}

type CallbackResponse struct {
	Status string `json:"status"`
}

// Callback is the callback endpoint for flip to hit when a payment data was changed
//
//encore:api public raw method=POST path=/payments/callback
func Callback(res http.ResponseWriter, req *http.Request) {
	var tx Transaction

	ctx := context.Background()
	// Start a database transaction
	dbTX, err := pgxDB.Begin(ctx)
	if err != nil {
		return
	}
	defer func(dbTX pgx.Tx, ctx context.Context) {
		err := dbTX.Rollback(ctx)
		if err != nil {
			rlog.Error("Error rolling back transaction", "err", err)
		}
	}(dbTX, ctx) // Ensure rollback in case of an error

	dataFormValue := req.PostFormValue("data")

	err = json.Unmarshal([]byte(dataFormValue), &tx)
	if err != nil {
		rlog.Error("Error unmarshalling JSON", "err", err)
		return
	}

	rollbackTickets := func(status string) {
		rlog.Error("Error: Payment failed", "status", status)

		for _, ticketID := range buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].TicketIDs {
			err := query.ChangeTicketsStatus(ctx, db.ChangeTicketsStatusParams{
				Status:   db.TicketStatusAvailable,
				TicketID: ticketID,
			})
			if err != nil {
				rlog.Error("Error: Error rolling back tickets status", status, err.Error())
				return
			}
		}

		rlog.Error("Error: Sold Ticket IDs", "rolled back", buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].TicketIDs)
		delete(buyTicketData, fmt.Sprintf("reserve:%d", tx.BillLinkID))
	}

	switch tx.Status {
	case "SUCCESSFUL":
		paymentData, err := json.Marshal(tx)
		_, err = query.InsertPayment(ctx, db.InsertPaymentParams{
			EventID:    buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].EventID,
			Data:       paymentData,
			Name:       tx.SenderName,
			Email:      tx.SenderEmail,
			BillLinkID: int32(tx.BillLinkID),
		})
		if err != nil {
			rlog.Error("Error: Error inserting payment: ", err.Error())
			return
		}

		for i, ticketID := range buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].TicketIDs {
			rlog.Info("Processing", "TicketId", ticketID)
			err := query.ChangeTicketsStatus(ctx, db.ChangeTicketsStatusParams{
				Status:   db.TicketStatusSold,
				TicketID: ticketID,
			})
			if err != nil {
				rlog.Error("Error: Error updating tickets status: ", tx.Status, err.Error())
				return

			}

			attendeeData, err := json.Marshal(buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].Attendees[i])
			_, err = query.InsertAttendee(ctx, db.InsertAttendeeParams{
				EventID:  buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].EventID,
				TicketID: ticketID,
				Data:     attendeeData,
			})
			if err != nil {
				rlog.Error("Error: Error inserting attendee: ", err.Error())
				return
			}
		}
		rlog.Info("Payment successful")
		delete(buyTicketData, fmt.Sprintf("reserve:%d", tx.BillLinkID))
		break
	case "FAILED":
		rollbackTickets(tx.Status)
		break
	case "CANCELLED":
		rollbackTickets(tx.Status)
		break
	default:
		log.Println("Error: Unknown payment status")
		rollbackTickets(tx.Status)
		break
	}

	// Commit the transaction if all tickets are deleted successfully
	if err := dbTX.Commit(ctx); err != nil {
		return
	}

	res.WriteHeader(http.StatusOK)
}
