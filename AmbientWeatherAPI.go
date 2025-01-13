package main

import (
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
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
		slog.Warn("An error occurred", "error", err)
		time.Sleep(time.Duration(5 * runs))
		return retryAPICall(runs)
	}
	defer resp.Body.Close()

	log.Println("response Status:", resp.Status)

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Error: received status code %d", resp.StatusCode)
		time.Sleep(time.Duration(5 * runs))
		return retryAPICall(runs)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("An error occurred", "error", err)
		time.Sleep(time.Duration(5 * runs))
		return retryAPICall(runs)
	}

	slog.Info(string(body))

	data := string(body)
	trimData := data[2 : len(data)-2]

	return trimData
}

func retryAPICall(runs int) string {
	if runs < 3 {
		return executeRequest(runs + 1)
	} else {
		slog.Error("Error: API call failed 3 times")
		return ""
	}
}
