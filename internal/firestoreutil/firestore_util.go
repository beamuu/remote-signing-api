package firestoreutil

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
)

// ---------- look-up ----------
func GetKMSKeyForAddress(ctx context.Context, c *firestore.Client, address string) (string, error) {
	doc, err := c.Collection("validator_keys").
		Doc(strings.ToLower(address)).
		Get(ctx)
	if err != nil {
		return "", errors.New("address not found in firestore")
	}
	v, err := doc.DataAt("kms_key")
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// ---------- save ----------
func SaveMapping(ctx context.Context, c *firestore.Client, address, keyPath string) error {
	_, err := c.Collection("validator_keys").
		Doc(strings.ToLower(address)).
		Set(ctx, map[string]interface{}{"kms_key": keyPath})
	return err
}
