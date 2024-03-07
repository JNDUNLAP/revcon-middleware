package handlers

import (
	"bytes"
	"context"
	"dunlap/app/log"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	SharedClient = &http.Client{Timeout: 300 * time.Second}
	MaxWorkers   = 5
)

type RequestProcessor struct {
	AccessToken string
	Headers     map[string]string
	Workers     int
}

type ResponseWithStopID struct {
	StopID   int               `json:"stopId"`
	Response []APIResponseItem `json:"response"`
	Error    string            `json:"error,omitempty"`
}

type FreightRequest struct {
	Requests []PayloadRequest `json:"requests"`
}

type PayloadRequest struct {
	StopId         int            `json:"stopId"`
	FreightDetails FreightDetails `json:"freightDetails"`
}

type FreightDetails struct {
	ConsigneeZip     string        `json:"consigneeZip"`
	ShipmentMode     string        `json:"shipmentMode"`
	ShipperZip       string        `json:"shipperZip"`
	Miles            string        `json:"miles"`
	ShipperCountry   string        `json:"shipperCountry"`
	ConsigneeCountry string        `json:"consigneeCountry"`
	EquipmentType    string        `json:"equipmentType"`
	Accessorials     []string      `json:"accessorials"`
	Items            []FreightItem `json:"items"`
}

type FreightItem struct {
	Class              string `json:"class"`
	IsHazardous        bool   `json:"isHazardous"`
	Pieces             int    `json:"pieces"`
	Weight             int    `json:"weight"`
	Packaging          string `json:"packaging"`
	NMFC               int    `json:"nmfc"`
	ProductDescription string `json:"productDescription"`
	Density            string `json:"density"`
	Length             int    `json:"length"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	Billed             int    `json:"billed"`
	Cost               int    `json:"cost"`
	UnitsWeight        string `json:"unitsWeight"`
	UnitsDensity       string `json:"unitsDensity"`
	UnitsDimension     string `json:"unitsDimension"`
}

type OrderProcessingError struct {
	Message string
}

type APIResponseItem struct {
	Name               string  `json:"name"`
	Scac               string  `json:"scac"`
	Billed             float64 `json:"billed"`
	TransitTime        string  `json:"transitTime"`
	BillToCode         *int    `json:"billToCode,omitempty"`
	ServiceType        string  `json:"serviceType"`
	ServiceDescription string  `json:"serviceDescription"`
}

func PostRequestWithContext(ctx context.Context, client *http.Client, url string, headers map[string]string, jsonPayload map[string]interface{}, stopID int) (string, error) {
	requestID := uuid.New().String()
	log.Info("POST %s: [UUID: %v] [StopID %d],  ", url, requestID, stopID)

	ctx = context.WithValue(ctx, "requestID", requestID)

	jsonData, err := json.Marshal(jsonPayload)
	if err != nil {
		log.Error("[StopID: %d] Error marshaling JSON: %v", stopID, err)
		// errorReturn := fmt.Sprintf("[StopID: %d] Error marshaling JSON: %v", stopID, err)
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		// errorReturn := fmt.Sprintf("Error posting Request", err)
		return "", err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("[StopID: %d] Error sending request: %v", stopID, err)
		// errorReturn := fmt.Sprintf("[StopID: %d] Error sending request: %v", stopID, err)
		return "", err
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Error("%d, REVCON RESPONSE: %s", resp.StatusCode, responseBody)

		// errorReturn := fmt.Sprintf("Error Reading Response: %s", err)
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
	    // You can log the response body for debugging or return it as part of the error message
	    log.Error("[UUID: %v] [StopID: %d] Non-200 HTTP status code: %v, Response Body: %s", requestID, stopID, resp.StatusCode, responseBody)
	    
	    // Here you could map status codes to custom error messages or take actions as needed
	    err = fmt.Errorf("non-200 HTTP status code received: %d, body: %s", resp.StatusCode, responseBody)
	    return "", err
	}
	
	log.Info("Status Code: %v, [UUID: %v] [StopID: %d] | Response %s", resp.StatusCode, requestID, stopID, responseBody)
	
	return string(responseBody), nil
}

func LoadJSONFile(filePath string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var payload map[string]interface{}
	err = json.NewDecoder(file).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func ParseRequests(r *http.Request) ([]PayloadRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var requests []PayloadRequest
	if err := json.Unmarshal(body, &requests); err != nil {
		return nil, err
	}
	return requests, nil
}

func NewRequestProcessor() (*RequestProcessor, error) {
	accessToken, err := GetOAuthToken()
	if err != nil {
		return nil, err
	}

	return &RequestProcessor{
		AccessToken: accessToken,
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", accessToken),
			"Content-Type":  "application/json",
		},
		Workers: MaxWorkers,
	}, nil
}

func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func ProcessSingleRequest(req PayloadRequest, headers map[string]string) (ResponseWithStopID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 69*time.Second)

	defer cancel()

	payloadMap := map[string]interface{}{
		"consigneeZip":     req.FreightDetails.ConsigneeZip,
		"shipmentMode":     req.FreightDetails.ShipmentMode,
		"shipperZip":       req.FreightDetails.ShipperZip,
		"miles":            req.FreightDetails.Miles,
		"shipperCountry":   req.FreightDetails.ShipperCountry,
		"consigneeCountry": req.FreightDetails.ConsigneeCountry,
		"equipmentType":    req.FreightDetails.EquipmentType,
		"accessorials":     req.FreightDetails.Accessorials,
		"items":            req.FreightDetails.Items,
	}

	response, err := PostRequestWithContext(ctx, SharedClient, os.Getenv("REVCON_API_URL"), headers, payloadMap, req.StopId)

	if err != nil {
		return ResponseWithStopID{
			StopID: req.StopId,
			Error:  fmt.Sprintf("Error Posting with Context: %s", err.Error()),
		}, nil
	}

	var apiResponse []APIResponseItem
	err = json.Unmarshal([]byte(response), &apiResponse)
	if err != nil {
		return ResponseWithStopID{
			StopID:   req.StopId,
			Response: []APIResponseItem{},
			Error:    err.Error(),
		}, nil
	}

	return ResponseWithStopID{StopID: req.StopId, Response: apiResponse}, nil
}

func (p *RequestProcessor) ProcessRequestsInParallel(requests []PayloadRequest) ([]ResponseWithStopID, error) {

	responseChan := make(chan ResponseWithStopID, len(requests))
	var wg sync.WaitGroup

	requestQueue := make(chan PayloadRequest, len(requests))
	for _, request := range requests {
		requestQueue <- request
	}
	close(requestQueue)

	for i := 0; i < p.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for req := range requestQueue {
				response, err := ProcessSingleRequest(req, p.Headers)
				if err != nil {
					log.Error("%v", err.Error())
					responseChan <- ResponseWithStopID{
						StopID:   req.StopId,
						Response: nil,
						Error:    err.Error(),
					}
					continue
				}
				responseChan <- response
			}
		}()
	}

	wg.Wait()
	close(responseChan)

	var responses []ResponseWithStopID

	for response := range responseChan {
		responses = append(responses, response)
	}

	return responses, nil
}

func SendJSONResponse(w http.ResponseWriter, responses []ResponseWithStopID) {

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(responses)
	if err != nil {
		log.Error("Error encoding JSON response: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
