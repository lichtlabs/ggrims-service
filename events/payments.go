package events

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
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
	Status                int    `json:"status"`
	Step                  int    `json:"step"`
	IsAddressRequired     int    `json:"is_address_required"`
	IsPhoneNumberRequired int    `json:"is_phone_number_required"`
}

// CreateBill creates a bill for users to pay
//
//encore:api private method=POST path=/payments
func CreateBill(ctx context.Context, req *CreateBillRequest) (*CreateBillResponse, error) {
	createBillEndpoint := fmt.Sprintf("%s/pwf/bill", secrets.FlipApiBaseEndpoint)

	// Prepare the data for the POST request
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
		fmt.Println("Error creating request:", err)
		return nil, errors.New("Error creating request")
	}

	// Set the appropriate headers
	reqs.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqs.Header.Set("Authorization", "Basic "+encodedCredentials)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(reqs)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, errors.New("Error making request")
	}
	defer resp.Body.Close()

	// Read and print the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return nil, errors.New("Error reading response")
	}
	fmt.Println("Response:", string(body))

	var jsonResponse CreateBillResponse
	json.Unmarshal(body, &jsonResponse)

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

type CallbackRequest struct {
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

type CallbackResponse struct {
	Status string `json:"status"`
}

// Callback is the callback endpoint for flip to hit when a payment data was changed
//
//encore:api public method=POST path=/payments/callback
func Callback(ctx context.Context, req *CallbackRequest) (*CallbackResponse, error) {
	log.Println("Name: ", req.SenderName)
	log.Println("Bank: ", req.SenderBank)
	log.Println("Email: ", req.SenderEmail)
	log.Println("Amount: ", req.Amount)
	log.Println("Status: ", req.Status)
	log.Println("Bank Type: ", req.SenderBankType)
	log.Println("Created At: ", req.CreatedAt)
	log.Println("Link ID: ", req.BillLinkID)

	rollbackTickets := func() {
		res, err := RollbackTickets(ctx, req.BillLinkID)
		if err != nil {
			log.Println("Error: Error rolling back tickets: ", err)
			return
		}
		log.Println("Error: Sold Ticket IDs: ", res.Data.TicketIDs)
	}

	switch req.Status {
	case "SUCCESSFUL":
		log.Println("Payment successful", req)
		res, err := ReserveTicket(ctx, req.BillLinkID)
		if err != nil {
			return nil, err
		}
		log.Println("Sold Ticket IDs: ", res.Data.TicketIDs)
		break
	case "FAILED":
		log.Println("Error: Payment failed", req)
		rollbackTickets()
		break
	case "CANCELLED":
		log.Println("Error: Payment cancelled", req)
		rollbackTickets()
		break
	default:
		log.Println("Error: Unknown payment status", req)
		// rollbackTickets()
		break
	}

	return &CallbackResponse{
		Status: "OK",
	}, nil
}

func encodeSecretKey() string {
	secretKey := secrets.FlipApiSecretKey

	// Encode to Base64
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(secretKey + ":"))

	// Print the Authorization header
	authorizationHeader := fmt.Sprintf("Basic %s", encodedAuth)
	fmt.Printf("%s\n", authorizationHeader)

	return authorizationHeader
}

var secrets struct {
	FlipApiBaseEndpoint string `json:"flip_api_base_endpoint"`
	FlipValidationToken string `json:"flip_validation_token"`
	FlipApiSecretKey    string `json:"flip_api_secret_key"`
}
