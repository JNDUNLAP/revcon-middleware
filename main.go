package main

import (
	"context"
	"dunlap/app/log"
	"dunlap/app/middleware"
	"dunlap/app/routes"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Error("Error loading .env file: %v", err)
		return
	}

	log.InitializeMongoDBLogger(true, 100)

	corsHandler := middleware.SetupCORS()

	r := mux.NewRouter()

	r.Use(middleware.ApiKeyMiddleware)
	r.Use(corsHandler.Handler)
	r.Use(middleware.RequestIDMiddleware)

	r.HandleFunc(os.Getenv("TOKEN_PATH"), routes.GetOAuthTokenHandler).Methods("POST")
	r.HandleFunc(os.Getenv("RATING_PATH"), routes.SubmitRatingHandler).Methods("POST")

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	server := &http.Server{
		Addr:         ":" + serverPort,
		Handler:      r,
		ReadTimeout:  1000 * time.Second,
		WriteTimeout: 1000 * time.Second,
	}

	go func() {
		log.Info("BatchGoBurr is running on port %s...", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("BatchGoBurr error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: %v", err)
	}

	log.Info("Server exiting")
}
