package handlers

import (
	"dunlap/app/log"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
)

func GetOAuthToken() (string, error) {
	data := url.Values{
		"client_id":     {os.Getenv("CLIENT_ID")},
		"client_secret": {os.Getenv("CLIENT_SECRET")},
		"grant_type":    {os.Getenv("GRANT_TYPE")},
	}

	resp, err := http.PostForm(os.Getenv("AUTH_URL"), data)
	if err != nil {
		log.Error("Posting to auth url: %v", err)
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error("non-OK HTTP status: %v", resp.Status)
		return "", err
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Error("Error decoding json %v", err)
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		log.Error("Problem gettting access token %v", err)

		return "", err
	}
	log.Info("Successfully got Auth Token")
	return token, nil
}
