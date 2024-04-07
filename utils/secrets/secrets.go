package secrets

import (
	"bytes"
	"fmt"
	"os"

	"github.com/loynoir/ExpandUser.go"
)

func GetSecretFromFile(path string) ([]byte, error) {
	var err error
	if path, err = ExpandUser.ExpandUser(path); err != nil {
		return []byte{}, fmt.Errorf("input `path` error: %v", err)
	}

	var rawContent []byte
	if rawContent, err = os.ReadFile(path); err != nil {
		return []byte{}, fmt.Errorf("input `path` error: %v", err)
	}

	return bytes.TrimSpace(rawContent), nil
}
