package main

/*
The AmbientWeatherAPI program provides a way to interact with the Ambient Weather API by making HTTP requests to
retrieve data from specific weather stations. The program handles the construction and execution of said API
requests, manages retries in case of errors, and logs the process for monitoring and debugging purposes.
*/
import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

const (
	URLBASE = "https://api.ambientweather.net/v1/devices/"
)

var (
	completeURL string
)

/*
The createURL function creates an HTTP URL to make API requests to the Ambient Weather API with the given API Key,
App Key, and MAC Address for a station.
*/
func createURL(macAddress string, apiKey string, appKey string) {
	completeURL = URLBASE + macAddress + "?apiKey=" + apiKey + "&applicationKey=" +
		appKey + "&limit=1&end_date=1723481785"
	slog.Info("URL Created: " + completeURL)
	return
}

/*
Executes the request to retrieve data for a given weather station, includes retry logic to manage errors and
http statuses.
- Sends an HTTP GET request to the specified `completeURL`.
- If an error occurs during the request, it retries using the `retryAPICall` function.
- Logs the HTTP response status for debugging purposes.
- If the response status code is not 200 (OK), it retries using the `retryAPICall` function.
- Reads and processes the response body:
  - If an error occurs while reading the body, it retries using `retryAPICall`.
  - Logs the response body and trims any unwanted characters before returning the processed data.
*/
func executeRequest(runs int) string {
	resp, err := http.Get(completeURL)
	if err != nil {
		return retryAPICall(runs, "Error occurred when trying to execute API request: "+err.Error())
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	slog.Info("Response Status:", "resp", resp.Status)
	if resp.StatusCode != http.StatusOK {
		return retryAPICall(runs, "Error: Received error status code "+strconv.Itoa(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return retryAPICall(runs, "Error occurred when trying read response: "+err.Error())
	}

	slog.Info(string(body))

	data := string(body)
	trimData := data[2 : len(data)-2]

	return trimData
}

/*
Handles Errors from the execute request, takes the error, number of runs performed, and a message.
If runs of the function reach or exceed 3 runs, then an error is logged, otherwise a warning is logged. Both the
warning and error log the error message and a message about the function. The program will wait based on the number of
runs starting from a 10-second wait to a 30-second wait. If an error is logged, the program returns a empty string
*/
func retryAPICall(runs int, info string) string {
	if runs < 3 {
		wait := 10 * runs
		slog.Warn("Warning #" + strconv.Itoa(runs) + ". Error: " + info + " retrying after " +
			strconv.Itoa(wait) + " second wait.")
		time.Sleep(time.Duration(wait) * time.Second)
		return executeRequest(runs + 1)
	} else {
		slog.Error("Error after 3 attempts: " + info + " returning back to caller method")
		return ""
	}
}
