package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
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

func initalizeSheet(runs int) {
	ctx := context.Background()

	credential, credErr := os.ReadFile("credentials.json")
	if credErr != nil {
		if errorHandler(credErr, runs, "Unable to read client secret file: ") {
			initalizeSheet(runs + 1)
		} else {
			return
		}
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, configErr := google.ConfigFromJSON(credential, "https://www.googleapis.com/auth/spreadsheets")
	if configErr != nil {
		if errorHandler(configErr, runs, "Unable to parse client secret file to config: ") {
			initalizeSheet(runs + 1)
		} else {
			return
		}
	}
	client := getClient(config)

	var serviceErr error
	service, serviceErr = sheets.NewService(ctx, option.WithHTTPClient(client))
	if serviceErr != nil {
		if errorHandler(serviceErr, runs, "Unable to retrieve Sheets client: ") {
			initalizeSheet(runs + 1)
		} else {
			return
		}

	}

	slog.Info("Successfully initialized Sheets client")
}

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
		slog.Warn("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		slog.Warn("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			return
		}
	}(f)

	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	slog.Info("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		slog.Error("Unable to cache oauth token: %v", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			slog.Error("Unable to cache oauth token: %v", err)
			return
		}
	}(f)
	jsonErr := json.NewEncoder(f).Encode(token)
	if jsonErr != nil {
		slog.Error("Unable to cache oauth token: %v", jsonErr)
		return
	}
}

func writeData(data string) {
	slog.Info("Data writing function...")

	year := time.Now().Year()
	writeRange := strconv.Itoa(time.Now().Year()) + "!A:A"

	response := getResponse(writeRange, strconv.Itoa(year), 1)
	if response == nil {
		slog.Error("Response from sheet is nil. Unable to write data.")
		return
	}
	sheetData := response.Values

	splitData := strings.Split(data, ",")
	emptyRow := len(sheetData) + 1

	slog.Info("Parsing through data...")
	var dataSheet [][]interface{}
	dataRow := make([]interface{}, len(allSensors))
	for _, item := range splitData {
		dataParts := strings.Split(item, ":")
		position := allSensors[strings.Trim(dataParts[0], "\"")].ID
		dataRow[stringToNum(position)] = dataParts[1]
	}

	dataSheet = append(dataSheet, dataRow)

	updateValues(strconv.Itoa(year), dataSheet, "!A"+strconv.Itoa(emptyRow), 0)

}

func updateValues(sheetName string, writeValues [][]interface{}, valuesRange string, runs int) {
	fullRange := sheetName + valuesRange
	body := &sheets.ValueRange{Values: writeValues}

	slog.Info("Updating values function. Writing to Range: " + valuesRange)

	slog.Info("Updating with Google API Client.")
	_, err := service.Spreadsheets.Values.Update(spreadsheetId, fullRange, body).
		ValueInputOption("RAW").Do()
	if err != nil {
		if errorHandler(err, runs, "Unable to update values in sheet: ") {
			updateValues(sheetName, writeValues, valuesRange, runs+1)
		} else {
			return
		}
	}

	slog.Info("Successfully updated values in sheet")
}

func getResponse(responseRange string, year string, runs int) *sheets.ValueRange {
	if !sheetExists(year, 1) {
		return nil
	}

	slog.Info("Getting Response from Sheet")
	resp, err := service.Spreadsheets.Values.Get(spreadsheetId, responseRange).Do()
	if err != nil {
		if errorHandler(err, runs, "Unable to retrieve data from sheet: ") {
			return getResponse(responseRange, year, runs+1)
		} else {
			return nil
		}
	}

	return resp
}

func sheetExists(sheetName string, runs int) bool {
	response, err := service.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		if errorHandler(err, runs, "Unable to retrieve data from sheet: ") {
			sheetExists(sheetName, runs+1)
		} else {
			return false
		}
	}

	for _, sheet := range response.Sheets {
		if sheet.Properties.Title == sheetName {
			return true
		}
	}
	slog.Info("Creating Sheet for Current Year")
	if createSheet(sheetName) {
		return true
	} else {
		return false
	}
}

func createSheet(sheetName string) bool {
	createRequest := &sheets.BatchUpdateSpreadsheetRequest{
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

	response := batchUpdateRequest(createRequest, 1)
	if response == nil {
		slog.Error("Unable to complete batch update request. Returning to previous function")
		return false
	}

	if len(response.Replies) > 0 && response.Replies[0].AddSheet != nil {
		slog.Info("Sheet created successfully", "sheetName", sheetName)

		slog.Info("Batch update request to freeze first row")

		freezeProperties := &sheets.SheetProperties{
			SheetId: response.Replies[0].AddSheet.Properties.SheetId,
			GridProperties: &sheets.GridProperties{
				FrozenRowCount: 1,
			},
		}

		freezeRequest := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				{
					UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
						Properties: freezeProperties,
						Fields:     "gridProperties.frozenRowCount",
					},
				},
			},
		}

		batchUpdateRequest(freezeRequest, 1)

		var sheetHeaders [][]interface{}

		headerRow := make([]interface{}, len(allSensors))
		for _, sensor := range allSensors {
			headerRow[stringToNum(sensor.ID)] = sensor.Description
		}

		sheetHeaders = append(sheetHeaders, headerRow)

		updateValues(sheetName, sheetHeaders, "!A1", 1)

		return true
	}
	slog.Error("Unable to complete batch update request. Returning to previous function")
	return false
}

func batchUpdateRequest(batchRequest *sheets.BatchUpdateSpreadsheetRequest, runs int) *sheets.BatchUpdateSpreadsheetResponse {
	var response *sheets.BatchUpdateSpreadsheetResponse = nil
	slog.Info("Requesting new batch update")
	response, err := service.Spreadsheets.BatchUpdate(spreadsheetId, batchRequest).Do()
	if err != nil {
		if errorHandler(err, runs, "Unable to complete batch update request: ") {
			return batchUpdateRequest(batchRequest, runs+1)
		} else {
			return nil
		}
	}
	return response
}

func stringToNum(letters string) int {
	result := 0
	for _, letter := range letters {
		currVal := int(letter-'A') + 1
		result = result*26 + currVal
	}
	return result - 1
}

func readSensors(runs int) {
	data, err := os.ReadFile("headers.txt")
	if err != nil {
		if errorHandler(err, runs, "Unable to read headers.txt") {
			readSensors(runs + 1)
		} else {
			return
		}
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}

		splitLine := strings.SplitN(line, ",", 3)
		if len(splitLine) < 3 {
			if errorHandler(err, runs, "Invalid line in headers.txt: "+line) {
				readSensors(runs + 1)
			} else {
				return
			}
		}
		sensor := SensorInfo{
			ID:          strings.TrimSpace(splitLine[1]),
			Description: strings.TrimSpace(splitLine[2]),
		}
		allSensors[splitLine[0]] = sensor

	}
}

func errorHandler(err error, runs int, message string) bool {
	if runs > 3 {
		slog.Error("Error after 3 attempts: " + message + err.Error() + " returning back to caller method")
		return false
	} else {
		wait := 10 * runs
		slog.Warn("Warning #" + strconv.Itoa(runs) + ". Error: " + message + err.Error() + " retrying after " +
			strconv.Itoa(wait) + " second wait.")
		time.Sleep(time.Duration(wait) * time.Second)
		return true
	}
}
