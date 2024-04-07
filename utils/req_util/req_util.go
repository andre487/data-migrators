package req_util

import (
	"io"
	"log"
)

func CloseBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Printf("WARN: Error when closing body: %v", err)
	}
}
