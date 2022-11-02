package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/cloudflare/cloudflare-go"
	"github.com/joho/godotenv"
	probing "github.com/prometheus-community/pro-bing"
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

var args struct {
	// Primary []string `arg:"-p, --primary, required, separate, env:PRIMARY_IPs" help:"Primary IPs (A record)"`
	// Backup  []string `arg:"-b, --backup, required, separate, env:BACKUP_IPs" help:"Backup IPs (A record)" `
	PrimaryAddresses   string `arg:"-p, --primary, env:PRIMARY_IPs" help:"Primary IP addresses to be created as A records - IPs should be comma-separated without spaces. Example: \"0.0.0.0,0.0.0.0\"" `
	BackupAddresses    string `arg:"-b, --backup, env:BACKUP_IPs" help:"Backup IP addresses to be created as A records - IPs should be comma-separated without spaces. Example: \"0.0.0.0,0.0.0.0\"" `
	CloudflareApiToken string `arg:"env:CLOUDFLARE_API_TOKEN, required" help:"Cloudflare API token"`
	CloudflareZoneID   string `arg:"env:CLOUDFLARE_ZONE_ID, required" help:"Cloudflare Zone ID"`
	RecordName         string `arg:"env:RECORD_NAME, required" help:"Record name to use. Example: \"foo.example.com\"" `

	LogLevel string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\" (default), \"warning\", \"error\", or \"fatal\"" `
}

func main() {
	godotenv.Load()
	arg.MustParse(&args)
	// Set logging
	// JSON Logger:
	// logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	// CLI Logger:
	// (.caller can display the line number)
	logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	if args.LogLevel == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if args.LogLevel == "info" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else if args.LogLevel == "warning" {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	} else if args.LogLevel == "error" {
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	} else if args.LogLevel == "fatal" {
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Output flags
	logger.Info().Msgf("Backup IPs: %v; Primary IPs %v", args.BackupAddresses, args.PrimaryAddresses)
	// Create an API client object
	cloudflareApi, err := cloudflare.NewWithAPIToken(args.CloudflareApiToken)
	checkNilErr(err)
	ctx := context.Background()

	userDetails, err := cloudflareApi.UserDetails(ctx)
	checkNilErr(err)
	logger.Info().Msgf("User Email: %v; ", userDetails.Email)
	zones, err := cloudflareApi.ListZones(ctx)

	checkNilErr(err)
	// Iterate over zones
	for _, zone := range zones {
		logger.Info().Msgf("Zone name: %v; Zone ID: %v", zone.Name, zone.ID)
	}
	zoneId := args.CloudflareZoneID
	if zoneId == "" {
		fmt.Print("Zone ID: ")
		zoneId = singleLineInput()
	} else {
		logger.Info().Msg("Using predefined zone ID")
	}

	// Creating a cloudflare.DNSRecord object can allow for filtering
	// Example: foo := cloudflare.DNSRecord{Name: "foo.example.com"}
	recordName := args.RecordName
	if recordName == "" {
		fmt.Print("Record Name (example: foo.example.com): ")
		recordName = singleLineInput()
	} else {
		logger.Info().Msg("Using predefined record name")
	}
	recordNameFilter := cloudflare.DNSRecord{Name: recordName, Type: "A"}
	records, err := cloudflareApi.DNSRecords(ctx, zoneId, recordNameFilter)
	checkNilErr(err)

	// Iterate over the filtered records and output their name, type, and value
	// Also create a list of the record contents (In this case, IPs)
	var ipContents []string
	for _, record := range records {
		logger.Info().Msgf("Record Name: %v; Record Type: %v; Record Value: %v", record.Name, record.Type, record.Content)
		ipContents = append(ipContents, record.Content)
	}
	// Create primaryAddresses and backupAddresses slices from the args
	primaryAddresses := strings.Split(args.PrimaryAddresses, ",")
	backupAddresses := strings.Split(args.BackupAddresses, ",")

	// Check what set of IPs are being used, primary or backup
	// var ipSet string
	if doAllElementsExist(ipContents, primaryAddresses) {
		logger.Info().Msg("Using primary IP set")
		// ipSet = "primary"
	} else if doAllElementsExist(ipContents, backupAddresses) {
		logger.Info().Msg("Using backup IP set")
		// ipSet = "backup"
	} else {
		logger.Info().Msg("Not using a known IP set")
		// ipSet = "unknown"
	}

	online := true
	// Ping the IPs to see if they are up, set online to false if none are up using ping()
	logger.Debug().Msgf("Primary Addresses: %v", primaryAddresses)
	for _, ip := range primaryAddresses {
		// for i := 1; i < 65535; i++  // TCP 0 is reserved, so start at 1
		logger.Info().Msgf("Pinging %v", ip)
		if ping(ip) {
			online = true
			break
		} else {
			online = false
		}
	}
	// if online {
	// logger.Info().Msg(color.InGreen("Online"))
	// } else {
	// logger.Info().Msg(color.InRed("Offline"))
	// }
	logger.Info().Msgf("Online: %v", online)
}

func ping(host string) bool {
	pinger, err := probing.NewPinger(host)
	checkNilErr(err)
	pinger.Count = 3
	pinger.Timeout = 5 * time.Second
	pinger.SetPrivileged(true)
	for {
		err = pinger.Run() // Blocks until finished.
		if err == nil {
			log.Debug().Msg("Ping sent successfully")
			break
		} else if err.Error() == "listen ip4:icmp : socket: operation not permitted" && runtime.GOOS == "linux" {
			logger.Error().
				Err(err).
				Str("help", "Run as root. For more information, check https://github.com/prometheus-community/pro-bing#linux").
				Msg("Privileged ping failed, attempting unprivileged ping")
			pinger.SetPrivileged(false)
			// No break here, so it will try again with unprivileged ping
		} else if err.Error() == "socket: permission denied" && runtime.GOOS == "linux" {
			logger.Fatal(). // Fatal because this is the last attempt at a ping
					Err(err).
					Str("help", "Privileged pings are disabled. To enable, run \"sudo sysctl -w net.ipv4.ping_group_range=\"0 2147483647\"\" For more information, check https://github.com/prometheus-community/pro-bing#linux").
					Msg("Privileged ping failed")
			break // This is unreachable, but the compiler doesn't know that
		} else {
			checkNilErr(err)
			break
		}
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	// Check if the server is online
	if stats.PacketsRecv > 0 {
		logger.Info().Msgf("Host %v is online", host)
		return true
	} else {
		logger.Info().Msgf("Host %v is offline", host)
		return false
	}
}

func doesElementExist(array []string, element any) bool {
	for _, v := range array {
		if v == element {
			return true
		}
	}
	return false
}

func doAllElementsExist(array []string, elements []string) bool {
	// Keep track of which elements exist in array, and which do not
	// If all elements exist, return true
	// If any element does not exist, return false
	for _, element := range elements {
		if !doesElementExist(array, element) {
			return false
		}
	}
	return true
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
