package events

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lichtlabs/ggrims-service/mail"
	mailtempl "github.com/lichtlabs/ggrims-service/mail/template"

	"encore.dev/beta/errs"
	"encore.dev/rlog"
	"github.com/lichtlabs/ggrims-service/events/db"
)

// CreateBillRequest represents the request parameters required to create a new bill.
type CreateBillRequest struct {
	Title                 string    `json:"title"`
	Amount                int       `json:"amount"`
	Type                  string    `json:"type"`
	ExpiredDate           time.Time 	`json:"expired_date"`
	RedirectURL           string    `json:"redirect_url"`
	IsAddressRequired     int       `json:"is_address_required"`
	IsPhoneNumberRequired int       `json:"is_phone_number_required"`
}

// CreateBillResponse represents the response structure for creating a bill.
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

// CreateBill creates a new bill with the given request parameters and returns the response or an error.
//
//encore:api public method=POST path=/payments
func CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	eb := errs.B()

	// Use either direct Flip URL or proxy URL based on configuration
	var createBillEndpoint string
	if secrets.ProxyBaseURL != "" {
		// Use proxy URL
		createBillEndpoint = fmt.Sprintf("%s/api/v2/pwf/bill", secrets.ProxyBaseURL)
	} else {
		// Use direct Flip URL
		createBillEndpoint = fmt.Sprintf("%s/pwf/bill", secrets.FlipApiBaseEndpoint)
	}

	// log the format of expired date
	rlog.Info("ExpiredDate: ", "format", req.ExpiredDate.Format("2006-01-02 15:04:05"))

	data := url.Values{}
	data.Set("title", req.Title)
	data.Set("amount", fmt.Sprintf("%d", req.Amount))
	data.Set("type", req.Type)
	data.Set("expired_date", req.ExpiredDate.Format("2006-01-02 15:04"))
	// data.Set("redirect_url", req.RedirectURL)
	data.Set("is_address_required", fmt.Sprintf("%d", req.IsAddressRequired))
	data.Set("is_phone_number_required", fmt.Sprintf("%d", req.IsPhoneNumberRequired))

	// print request data
	rlog.Info("CreateBillRequest: ", "data", data.Encode())

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
	rlog.Info("Flip is responding:", "FlipResponse", string(body))

	var jsonResponse CreateBillResponse
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		rlog.Info("Error unmarshalling JSON:", "err", err)
		return nil, err
	}
	rlog.Info("CreateBillResponse: ", "jsonResponse", jsonResponse)

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

// Callback handles the HTTP request for processing a payment callback, updating the database, and changing ticket statuses.
//
//encore:api public raw method=POST path=/payments/callback
func Callback(res http.ResponseWriter, req *http.Request) {
	var tx Transaction

	ctx := context.Background()
	// Start a database transaction
	dbTX, err := pgxDB.Begin(ctx)
	if err != nil {
		rlog.Error("Failed to start transaction", "err", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	var committed bool
	defer func() {
		if !committed {
			err := dbTX.Rollback(ctx)
			if err != nil && err != pgx.ErrTxClosed {
				rlog.Error("Error rolling back database transaction", "err", err)
			}
		}
	}()

	dataFormValue := req.PostFormValue("data")
	log.Println("dataFormValue", dataFormValue)

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
		if err != nil {
			rlog.Error("Error: Error marshalling payment data: ", err.Error())
			return
		}

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

		ticketPrice := 0

		for _, ticketID := range buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].TicketIDs {
			if ticketPrice == 0 {
				ticket, err := query.GetTicket(ctx, ticketID)
				if err != nil {
					rlog.Error("Error: Error getting ticket: ", err.Error())
					return
				}
				log.Println(ticket.Price)
				ticketPrice, err = strconv.Atoi(ticket.Price)
				if err != nil {
					rlog.Error("Error: Error converting ticket price to int: ", err.Error())
					return
				}
			}

			rlog.Info("Processing", "TicketId", ticketID)
			err = query.ChangeTicketsStatus(ctx, db.ChangeTicketsStatusParams{
				Status:   db.TicketStatusSold,
				TicketID: ticketID,
			})

			for j := range buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].Attendees {
				attendeeData, err := json.Marshal(buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].Attendees[j])
				if err != nil {
					rlog.Error("Error: Error marshalling attendee data: ", err.Error())
					return
				}

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
			if err != nil {
				rlog.Error("Error: Error updating tickets status: ", tx.Status, err.Error())
				return

			}
		}
		rlog.Info("Payment successful")

		var buff bytes.Buffer
		ctx := context.Background()
		err = mailtempl.PurchaseConfirmationEmail(mailtempl.PurchaseConfirmation{
			CustomerName: tx.SenderName,
			ItemName:     tx.BillTitle,
			ItemPrice:    strconv.Itoa(ticketPrice),
			TotalPrice:   strconv.Itoa(tx.Amount),
			OrderNumber:  tx.ID,
		}).Render(ctx, &buff)
		if err != nil {
			rlog.Error("Error: Error rendering purchase confirmation email: ", err.Error())
			return
		}

		body := buff.String()
		err = mail.SendTicketMail(ctx, &mail.SendTicketMailRequest{
			Recipients:   []string{tx.SenderEmail},
			TicketHashes: buyTicketData[fmt.Sprintf("reserve:%d", tx.BillLinkID)].TicketHashes,
			Body:         body,
		})
		if err != nil {
			rlog.Error("Error: Error sending ticket mail: ", err.Error())
			return
		}

		delete(buyTicketData, fmt.Sprintf("reserve:%d", tx.BillLinkID))
	case "FAILED":
		rollbackTickets(tx.Status)
	case "CANCELLED":
		rollbackTickets(tx.Status)
	case "EXPIRED":
		rollbackTickets(tx.Status)
	default:
		rlog.Error("Error: Unknown payment status", "status", tx.Status)
		rollbackTickets(tx.Status)
	}

	// Commit the transaction if all operations are successful
	if err := dbTX.Commit(ctx); err != nil {
		rlog.Error("Failed to commit transaction", "err", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	committed = true

	res.WriteHeader(http.StatusOK)
}
