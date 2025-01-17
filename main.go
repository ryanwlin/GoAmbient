package main

/*
Main file that initializes and schedules periodic API calls to the AmbientWeather API, receiving data from a
specified weather station, provided through a MAC Address, API Key, and Application Key. The retrieved data from the
API is then written to a Google Sheet through the Sheets.go program. The main program runs continuously, calling the
AmbientWeather API every 5 minutes.
*/
import (
	"log/slog"
	"os"
	"strings"
	"time"
)

/*
Main function that initializes all necessary functions like the Google Sheets Service and the Ambient Weather API
by providing secrets like the API Key, APP Key, and MAC Address to build the HTTP to retrieve data from API calls.
*/
func main() {
	slog.Info("Start program at", "time", time.Now())

	slog.Info("Initializing Sheets")
	initializeSheet(1) //Initialize the Google Sheet Service
	readSensors(1)     //Reads all sensor descriptions from headers.txt and stores them in a map

	//Retries secrets from secrets.txt file, will restive from K8s after setup
	secretFile, err := os.ReadFile("secrets.txt")
	if err != nil {
		slog.Warn("Unable to read headers.txt: %v", err)
	}
	secret := strings.Split(string(secretFile), ",")

	createURL(secret[0], secret[1], secret[2]) //Creates URL to call Ambient Weather API, with all the provided secrets

	slog.Info("Starting scheduled API calls")
	scheduleAPI()

}

/*
Function that schedules calls to retrieve data from the Ambient Weather API every 5 minutes. Once data is retrieved
a function in Sheets.go is called to write the data to a Google Sheet.
*/
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
	scheduleAPI() //Recalls function to schedule and run API calls
}
