package mail

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"encore.dev/rlog"
	gomail "gopkg.in/gomail.v2"
)

var SenderName = fmt.Sprintf(
	"%s <%s>",
	secrets.SenderName,
	secrets.AuthEmail,
)

type SendTicketMailRequest struct {
	Body         string
	TicketHashes []string
	Recipients   []string
}

// SendTicketMail sends an email to specified recipients with optional TicketHashes based on SendTicketMailRequest.
//
//encore:api private
func SendTicketMail(ctx context.Context, req *SendTicketMailRequest) error {
	mailer := gomail.NewMessage()
	mailer.SetHeader("From", SenderName)
	mailer.SetHeader("To", req.Recipients...)
	mailer.SetAddressHeader("Cc", secrets.AdminMail, "Licht Labs Admin")
	mailer.SetHeader("Subject", "Your ticket is here!")
	mailer.SetBody("text/html", req.Body)

	createdFiles := []string{}
	if len(req.TicketHashes) > 0 {
		// write file to temporary dir
		for i, hash := range req.TicketHashes {
			qrcode := genTicketQR(hash)
			filename := fmt.Sprintf("/tmp/%s-%d.png", req.Recipients[0], i)
			err := os.WriteFile(filename, qrcode, 0644)
			if err != nil {
				rlog.Error("Failed to write qrcode to file", "error", err)
				return err
			}
			mailer.Attach(filename)
			createdFiles = append(createdFiles, filename)
		}
	}

	smtpPort, err := strconv.Atoi(secrets.SmtpPort)
	if err != nil {
		rlog.Error("Failed to parse smtp port", "error", err)
		return err
	}

	dialer := gomail.NewDialer(
		secrets.SmtpHost,
		smtpPort,
		secrets.AuthEmail,
		secrets.AuthPassword,
	)

	err = dialer.DialAndSend(mailer)
	if err != nil {
		rlog.Error("Failed to send mail", "error", err)
		return err
	}

	// delete all temporary created files
	for _, file := range createdFiles {
		err := os.Remove(file)
		if err != nil {
			rlog.Error("Failed to delete temporary file", "error", err)
		}
	}

	return nil
}

var secrets struct{ AdminMail, AuthEmail, AuthPassword, SmtpHost, SmtpPort, SenderName string }
