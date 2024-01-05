package routes

import (
	"dunlap/app/handlers"
	"dunlap/app/log"
	"fmt"
	"net/http"
	"time"
)

func SubmitRatingHandler(w http.ResponseWriter, r *http.Request) {

	startTime := time.Now()

	requests, err := handlers.ParseRequests(r)
	if err != nil {
		parsingError := fmt.Sprintf("Error Parsing Requests: %s", err)
		handlers.RespondWithError(w, http.StatusBadRequest, parsingError)
		return
	}

	processor, err := handlers.NewRequestProcessor()

	if err != nil {
		requestError := fmt.Sprintf("Error Handling Requests: %s", err)
		handlers.RespondWithError(w, http.StatusBadRequest, requestError)
		return
	}

	responses, err := processor.ProcessRequestsInParallel(requests)

	if err != nil {
		conncurencyError := fmt.Sprintf("Error Handling Requests: %s", err)
		handlers.RespondWithError(w, http.StatusInternalServerError, conncurencyError)
		return
	}

	handlers.SendJSONResponse(w, responses)

	duration := time.Since(startTime)
	log.Info("Request completed in %.2f seconds", duration.Seconds())

}
