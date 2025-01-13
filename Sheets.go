package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type SensorInfo struct {
	ID          string
	Description string
}

var (
	service       *sheets.Service = nil
	spreadsheetId                 = "1XfM5AjJzs8rEJ9PDDi9N0DEPOqw-P1RYdM4ST8Ga4uM"
	allSensors                    = make(map[string]SensorInfo)
)

func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	slog.Info("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	slog.Info("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func initalizeSheet(runs int) {
	ctx := context.Background()

	credential, err := os.ReadFile("credentials.json")
	if err != nil {
		slog.Warn("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(credential, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		slog.Warn("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	service, err = sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		slog.Warn("Unable to retrieve Sheets client: %v", err)
	}

	slog.Info("Successfully retrieved Sheets client")
}

func writeData(data string) {
	if service == nil {
		initalizeSheet(1)
	}

	year := time.Now().Year()
	writeRange := strconv.Itoa(year) + "!A:A"

	response := getResponse(writeRange, strconv.Itoa(year), 1)
	sheetData := response.Values

	splitData := strings.Split(data, ",")

	for _, item := range splitData {
		slog.Info(item)
	}

	emptyRow := len(sheetData) + 1

	updateValues(strconv.Itoa(year), sheetData, writeRange+strconv.Itoa(emptyRow), 0)

}

func updateValues(sheetName string, values [][]interface{}, valuesRange string, runs int) {

}

func getResponse(responseRange string, year string, runs int) *sheets.ValueRange {
	sheetExists(year)

	slog.Info("Getting Info from Sheet")
	resp, err := service.Spreadsheets.Values.Get(spreadsheetId, responseRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	return resp
}

func sheetExists(sheetName string) {
	response, err := service.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		slog.Warn("Unable to retrieve data from sheet: %v", err)
	}

	for _, sheet := range response.Sheets {
		if sheet.Properties.Title == sheetName {
			return
		}
	}

	slog.Info("Creating Sheet for Current Year")
	createSheet(sheetName)

}

func createSheet(sheetName string) {
	request := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheetName,
					},
				},
			},
		},
	}

	_, err := service.Spreadsheets.BatchUpdate(spreadsheetId, request).Do()
	if err != nil {
		log.Fatalf("Unable to create sheet: %v", err)
	}
	slog.Info("Sheet created successfully", "sheetName", sheetName)
}

func stringToNum(letters string) int {
	result := 0
	for _, letter := range letters {
		currVal := int(letter-'A') + 1
		result = result*26 + currVal
	}
	return result
}

func readSensors() {
	data, err := ioutil.ReadFile("headers.txt")
	if err != nil {
		slog.Warn("Unable to read headers.txt: %v", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}

		splitLine := strings.SplitN(line, ",", 3)
		if len(splitLine) < 3 {
			slog.Warn("Invalid line in headers.txt: %v", line)
		}
		sensor := SensorInfo{
			ID:          strings.TrimSpace(splitLine[1]),
			Description: strings.TrimSpace(splitLine[2]),
		}
		allSensors[splitLine[0]] = sensor

	}
}
