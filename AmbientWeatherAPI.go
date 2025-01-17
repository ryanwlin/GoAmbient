package main

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

func creatURL(macAddress string, apiKey string, appKey string) {
	completeURL = URLBASE + macAddress + "?apiKey=" + apiKey + "&applicationKey=" + appKey + "&limit=1&end_date=1723481785"
	slog.Info("URL Created: " + completeURL)
	return
}

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
