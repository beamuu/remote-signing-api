package kmsutil

import (
	"encoding/pem"
	"errors"
)

func pemToDER(pemStr string) ([]byte, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	return block.Bytes, nil
}
