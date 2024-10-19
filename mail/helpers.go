package mail

import (
	"fmt"
	"github.com/skip2/go-qrcode"
	"log"
)

// BaseURL is the base URL for calling the Encore application's API.
type BaseURL string

const Local BaseURL = "http://localhost:4000"

// Environment returns a BaseURL for calling the cloud environment with the given name.
func Environment(name *string) BaseURL {
	if name == nil || *name == "" {
		return Local
	}

	return BaseURL(fmt.Sprintf("https://%s-ggrims-services-xixi.encr.app", name))
}

// genTicketQR generates a QR code in PNG format for the given hash with a predefined URL and returns the PNG byte array.
func genTicketQR(hash string) []byte {
	var png []byte

	url := fmt.Sprintf("%s/scans/%s", Environment(nil), hash)
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		log.Println("Error generating QR code: ", err)
	}

	return png
}
