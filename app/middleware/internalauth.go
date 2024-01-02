package middleware

import (
	"dunlap/app/log"
	"dunlap/app/mongo"
	"net/http"
	"os"
	"strings"
)

func ApiKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Error("No Authorization header provided")
			http.Error(w, "Unauthorized - No API Key provided", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == authHeader {
			log.Error("Malformed Authorization header")
			http.Error(w, "Unauthorized - Malformed Authorization header", http.StatusUnauthorized)
			return
		}

		if !mongo.ValidateMongoKey(os.Getenv("MongoURI"), "honda", "apikeys", token) {
			log.Error("Invalid API Key")
			http.Error(w, "Unauthorized - Invalid API Key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
