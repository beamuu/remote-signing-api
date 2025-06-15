package handler

import (
	"encoding/hex"
	"remote-signing-api/internal/kmsutil"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type SignRequest struct {
	Address string `json:"address"`
	Hash    string `json:"hash"`
}
type SignResponse struct {
	Signature string `json:"signature"`
}

func NewSignHandler(svc *kmsutil.SignerService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req SignRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("invalid json")
		}
		hash, err := hex.DecodeString(strings.TrimPrefix(req.Hash, "0x"))
		if err != nil || len(hash) != 32 {
			return c.Status(fiber.StatusBadRequest).SendString("hash must be 32-byte hex")
		}

		sig, err := svc.SignWithKMS(c.Context(), req.Address, hash)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.JSON(SignResponse{Signature: sig})
	}
}
