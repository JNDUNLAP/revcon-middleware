package middleware

import (
	"os"

	"github.com/rs/cors"
)

func SetupCORS() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins: []string{os.Getenv("CORS_ALLOWED_ORIGINS")},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	})
}
