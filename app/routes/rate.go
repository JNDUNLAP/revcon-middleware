package routes

import (
	"dunlap/app/handlers"
	"dunlap/app/log"
	"net/http"
	"time"
)

func SubmitRatingHandler(w http.ResponseWriter, r *http.Request) {

	startTime := time.Now()

	requests, err := handlers.ParseRequests(r)
	if err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "Error parsing requests")
		return
	}

	processor, err := handlers.NewRequestProcessor()

	if err != nil {
		handlers.RespondWithError(w, http.StatusBadRequest, "Error Intitating Workers")
		return
	}

	responses, err := processor.ProcessRequestsInParallel(requests)

	if err != nil {
		handlers.RespondWithError(w, http.StatusInternalServerError, "Error processing requests")
		return
	}

	handlers.SendJSONResponse(w, responses)

	duration := time.Since(startTime)
	log.Info("Request completed in %.2f seconds", duration.Seconds())

}
