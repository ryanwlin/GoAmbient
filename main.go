package main

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

func main() {
	readSensors(1)
	slog.Info("Start program at", "time", time.Now())

	slog.Info("Initializing Sheets")
	initalizeSheet(1)

	secretFile, err := os.ReadFile("secrets.txt")
	if err != nil {
		slog.Warn("Unable to read headers.txt: %v", err)
	}
	secret := strings.Split(string(secretFile), ",")

	creatURL(secret[0], secret[1], secret[2])
	slog.Info("Starting scheduled API calls")

	scheduleAPI()

}

func scheduleAPI() {
	currentTime := time.Now()

	nextRun := currentTime.Truncate(time.Minute).Add(5 * time.Minute)
	nextRun = nextRun.Truncate(5 * time.Minute)
	waitDuration := time.Until(nextRun)
	slog.Info("Next API call scheduled at:", "time", nextRun)

	time.Sleep(waitDuration)
	slog.Info("API Function called at: ", "time", time.Now())
	data := executeRequest(0)
	if data == "" {
		slog.Error("API request resulted in empty values")
	}
	writeData(data)

	scheduleAPI()
}
