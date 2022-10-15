package main

import (
	"context"
	"os"
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
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.DebugLevel).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Create an API client object
	cloudflareApi, err := cloudflare.NewWithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))
	checkNilErr(err)
	ctx := context.Background()
	userDetails, err := cloudflareApi.UserDetails(ctx)
	checkNilErr(err)
	logger.Info().Msgf("User Email: %v\n", userDetails.Email)
}

func checkNilErr(err error) {
	if err != nil {
		// log.Fatalln("Error:\n%v\n", err)
		logger.Error().
			Err(err).
			Msg("something happened!")
	}
}
