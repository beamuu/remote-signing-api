package handler

import (
	"remote-signing-api/internal/kmsutil"

	"github.com/gofiber/fiber/v2"
)

type OnboardRequest struct {
	CustomerID string `json:"customer_id"` // free-form, must be unique per user
}
type OnboardResponse struct {
	Address string `json:"address"`
}

func NewOnboardHandler(svc *kmsutil.SignerService, location, keyRing string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req OnboardRequest
		if err := c.BodyParser(&req); err != nil || req.CustomerID == "" {
			return c.Status(fiber.StatusBadRequest).SendString("customer_id required")
		}

		addr, err := svc.CreateCustomerKey(c.Context(), location, keyRing, req.CustomerID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.JSON(OnboardResponse{Address: addr})
	}
}
