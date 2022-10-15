package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Set loggers
// var (
//
//	InfoLogger    = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)
//	WarningLogger = log.New(os.Stdout, "WARNING: ", log.LstdFlags|log.Lshortfile)
//	ErrorLogger   = log.New(os.Stdout, "ERROR: ", log.LstdFlags|log.Lshortfile)
//
// )
var logger zerolog.Logger

func main() {
	godotenv.Load()
	// Set logging
	// JSON Logger
	// logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	// CLI Logger
	// .caller can display the line number
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.DebugLevel).With().Timestamp().Caller().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Create an API client object
	cloudflareApi, err := cloudflare.NewWithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))
	checkNilErr(err)
	ctx := context.Background()

	userDetails, err := cloudflareApi.UserDetails(ctx)
	checkNilErr(err)
	logger.Info().Msgf("User Email: %v; ", userDetails.Email)
	zones, err := cloudflareApi.ListZones(ctx)

	checkNilErr(err)
	// Iterate over zones
	for index, zone := range zones {
		logger.Info().Msgf("Zone name: %v; Zone ID: %v; Index: %v", zone.Name, zone.ID, index)
	}
	zoneId := os.Getenv("CLOUDFLARE_ZONE_ID")
	if zoneId == "" {
		fmt.Print("Zone ID: ")
		zoneId = singleLineInput()
	} else {
		logger.Info().Msg("Using zone ID from enviroment variable")
	}

	// Creating a cloudflare.DNSRecord object can allow for filtering
	// Example: foo := cloudflare.DNSRecord{Name: "foo.example.com"}
	recordName := os.Getenv("RECORD_NAME")
	if recordName == "" {
		fmt.Print("Record Name (example: foo.example.com): ")
		recordName = singleLineInput()
	} else {
		logger.Info().Msg("Using record name from enviroment variable")
	}
	recordNameFilter := cloudflare.DNSRecord{Name: recordName}
	records, err := cloudflareApi.DNSRecords(ctx, zoneId, recordNameFilter)
	checkNilErr(err)

	// Iterate over records and output their name, type, and value
	for _, record := range records {
		logger.Info().Msgf("Record Name: %v; Record Type: %v; Record Value: %v", record.Name, record.Type, record.Content)
	}
}

func checkNilErr(err error) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		logger.Error().
			Err(err).
			Msg("something happened!")
	}
}
func singleLineInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	checkNilErr(err)
	input = strings.TrimSpace(input)
	// fmt.Print("\n")
	return input
}
