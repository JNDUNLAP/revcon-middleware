package routes

import (
	"dunlap/app/handlers"
	"dunlap/app/log"
	"encoding/json"
	"net/http"
)

type TokenResponse struct {
	Token string `json:"token"`
}

func GetOAuthTokenHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := handlers.GetOAuthToken()
	if err != nil {
		log.Error("Problem with auth Function %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := TokenResponse{Token: accessToken}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

}
