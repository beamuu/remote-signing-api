package main

import (
	"context"
	"log"
	"os"

	"remote-signing-api/internal/handler"
	"remote-signing-api/internal/kmsutil"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// ---- Config via env vars ----
	projectID := mustEnv("GCP_PROJECT_ID")
	location := mustEnv("KMS_LOCATION")  // e.g. "global"
	keyRingID := mustEnv("KMS_KEY_RING") // e.g. "remote-signing-api"

	ctx := context.Background()
	svc, err := kmsutil.NewSignerService(ctx, projectID)
	if err != nil {
		log.Fatalf("init signer service: %v", err)
	}
	defer svc.Close()

	app := fiber.New()

	app.Post("/sign", handler.NewSignHandler(svc))
	app.Post("/onboard", handler.NewOnboardHandler(svc, location, keyRingID))

	log.Println("API listening on :8080")
	log.Fatal(app.Listen(":8080"))
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env %s", k)
	}
	return v
}
