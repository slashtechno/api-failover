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
	"github.com/sirupsen/logrus"
)

// Set loggers
// var (
//
//	InfoLogger    = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)
//	WarningLogger = log.New(os.Stdout, "WARNING: ", log.LstdFlags|log.Lshortfile)
//	ErrorLogger   = log.New(os.Stdout, "ERROR: ", log.LstdFlags|log.Lshortfile)
//
// )

var args struct {
	// Primary []string `arg:"-p, --primary, required, separate, env:PRIMARY_IPs" help:"Primary IPs (A record)"`
	// Backup  []string `arg:"-b, --backup, required, separate, env:BACKUP_IPs" help:"Backup IPs (A record)" `
	PrimaryHosts       string `arg:"-p, --primary, env:PRIMARY_IPs" help:"Primary hosts to be created as A records - IPs should be comma-separated without spaces. Example: \"0.0.0.0,0.0.0.0\"" `
	BackupHosts        string `arg:"-b, --backup, env:BACKUP_IPs" help:"Backup hosts to be created as A records - IPs should be comma-separated without spaces. Example: \"0.0.0.0,0.0.0.0\"" `
	CloudflareApiToken string `arg:"env:CLOUDFLARE_API_TOKEN, required" help:"Cloudflare API token"`
	CloudflareZoneID   string `arg:"env:CLOUDFLARE_ZONE_ID, required" help:"Cloudflare Zone ID"`
	RecordName         string `arg:"env:RECORD_NAME, required" help:"Record name to use. Example: \"foo.example.com\"" `
	ProxyRecords       bool   `arg:"env:PROXY_RECORDS" help:"Proxy records" default:"true"`

	LogLevel        string `arg:"--log-level, env:LOG_LEVEL" help:"\"debug\", \"info\" (default), \"warning\", \"error\", or \"fatal\"" ` // ~~Could~~ Should add default value
	ForceLogColor   bool   `arg:"--force-log-color, env:FORCE_LOG_COLOR" help:"Force colored logs" default:"true"`
	DisableLogColor bool   `arg:"--disable-log-color, env:DISABLE_LOG_COLOR" help:"Disable colored logs - Overrides --force-log-color" `

	LoopProgram bool `arg:"-l, --loop-program, env:LOOP_PROGRAM" help:"Loop program" default:"false"`
}

type HostSet struct {
	Primary struct {
		Hosts       []string `json:"hosts"`
		OnlineHosts []string `json:"OnlineHosts"`
	} `json:"primary"`
	Backup struct {
		Hosts       []string `json:"hosts"`
		OnlineHosts []string `json:"OnlineHosts"`
	} `json:"backup"`
}

func main() {
	godotenv.Load()
	arg.MustParse(&args)

	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{PadLevelText: true, DisableQuote: true, ForceColors: args.ForceLogColor, DisableColors: args.DisableLogColor})

	if args.LogLevel == "debug" {
		logrus.SetLevel(logrus.DebugLevel)
	} else if args.LogLevel == "info" {
		logrus.SetLevel(logrus.InfoLevel)
	} else if args.LogLevel == "warning" {
		logrus.SetLevel(logrus.WarnLevel)
	} else if args.LogLevel == "error" {
		logrus.SetLevel(logrus.ErrorLevel)
	} else if args.LogLevel == "fatal" {
		logrus.SetLevel(logrus.FatalLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Output configuation
	logrus.Infof("Backup hosts: %v; Primary hosts %v", args.BackupHosts, args.PrimaryHosts)
	// Create an API client object
	cloudflareApi, err := cloudflare.NewWithAPIToken(args.CloudflareApiToken)
	checkNilErr(err)
	ctx := context.Background()

	userDetails, err := cloudflareApi.UserDetails(ctx)
	checkNilErr(err)
	logrus.Infof("User Email: %v; ", userDetails.Email)
	zones, err := cloudflareApi.ListZones(ctx)

	checkNilErr(err)
	// Iterate over zones
	for _, zone := range zones {
		logrus.Infof("Zone name: %v; Zone ID: %v", zone.Name, zone.ID)
	}
	zoneId := args.CloudflareZoneID
	if zoneId == "" {
		fmt.Print("Zone ID: ")
		zoneId = singleLineInput()
	} else {
		logrus.Info("Using predefined zone ID")
	}
	for {

		// Creating a cloudflare.DNSRecord object can allow for filtering
		// Example: foo := cloudflare.DNSRecord{Name: "foo.example.com"}
		recordName := args.RecordName
		if recordName == "" {
			fmt.Print("Record Name (example: foo.example.com): ")
			recordName = singleLineInput()
		} else {
			logrus.Info("Using predefined record name")
		}
		recordNameFilter := cloudflare.DNSRecord{Name: recordName, Type: "A"}
		records, err := cloudflareApi.DNSRecords(ctx, zoneId, recordNameFilter)
		checkNilErr(err)

		// Iterate over the filtered records and output their name, type, and value
		// Also create a list of the record contents (In this case, IPs)
		var ipContents []string
		for _, record := range records {
			logrus.Infof("Record Name: %v; Record Type: %v; Record Value: %v", record.Name, record.Type, record.Content)
			ipContents = append(ipContents, record.Content)
		}

		// Create a HostSet object
		hostSet := HostSet{}
		hostSet.Primary.Hosts = strings.Split(args.PrimaryHosts, ",")
		hostSet.Backup.Hosts = strings.Split(args.BackupHosts, ",")

		// Check what set of hosts is being used, primary or backup
		var ipSet string
		if doAllElementsExist(ipContents, hostSet.Primary.Hosts) {
			logrus.Info("Using primary IP set")
			ipSet = "primary"
		} else if doAllElementsExist(ipContents, hostSet.Backup.Hosts) {
			logrus.Info("Using backup IP set")
			ipSet = "backup"
		} else {
			logrus.Info("Not using a known IP set")
			ipSet = "unknown"
		}

		// Ping the hosts and append the online hosts to the HostSet object
		// Ping primary hosts
		logrus.Debugf("Primary hosts: %v", hostSet.Primary.Hosts)
		for _, ip := range hostSet.Primary.Hosts {
			logrus.Infof("Pinging %v", ip)
			if ping(ip) {
				// Append the IP to the list of online hosts
				hostSet.Primary.OnlineHosts = append(hostSet.Primary.OnlineHosts, ip)
			}
		}
		logrus.Infof("Online primary hosts: %v", hostSet.Primary.OnlineHosts)
		// Ping backup hosts
		logrus.Debugf("Backup hosts: %v", hostSet.Backup.Hosts)
		for _, ip := range hostSet.Backup.Hosts {
			logrus.Infof("Pinging %v", ip)
			if ping(ip) {
				// Append the IP to the list of online hosts
				hostSet.Backup.OnlineHosts = append(hostSet.Backup.OnlineHosts, ip)
			}
		}
		logrus.Infof("Online backup hosts: %v", len(hostSet.Backup.OnlineHosts))

		// Only switch records if both sets of hosts have been specified
		if args.BackupHosts != "" && args.PrimaryHosts != "" {
			if len(hostSet.Primary.OnlineHosts) > 0 && ipSet == "primary" {
				logrus.Debugf("Primary hosts: %v\nPrimary hosts length %v", hostSet.Primary.Hosts, len(hostSet.Primary.Hosts))
				logrus.Info("Primary hosts are up and record is set to primary, doing nothing")
			} else if len(hostSet.Primary.OnlineHosts) > 0 && ipSet == "backup" {
				// If at least one primary host is up and the records are set to backup hosts, update the records to the primary hosts

				// If the number of current records is equal to the number of primary hosts, update the records
				if len(records) == len(hostSet.Primary.Hosts) {
					logrus.Info("Updating records to primary IPs")
					for i, record := range records {
						// Update the record variable
						record.Content = hostSet.Primary.Hosts[i]
						record.Proxied = &args.ProxyRecords
						// Update the record on Cloudflare
						err = cloudflareApi.UpdateDNSRecord(ctx, zoneId, record.ID, record)
						checkNilErr(err)
					}
				} else {
					// If the number of current records is not equal to the number of primary hosts, delete all records and create new ones
					logrus.Info("Deleting records and creating new ones")
					for _, record := range records {
						err = cloudflareApi.DeleteDNSRecord(ctx, zoneId, record.ID)
						checkNilErr(err)
					}
					for _, ip := range hostSet.Primary.Hosts {
						record := cloudflare.DNSRecord{
							Type:    "A",
							Name:    recordName,
							Content: ip,
							TTL:     1,
							Proxied: &args.ProxyRecords,
						}
						_, err = cloudflareApi.CreateDNSRecord(ctx, zoneId, record)
						checkNilErr(err)
					}
				}
			} else if len(hostSet.Primary.OnlineHosts) == 0 && ipSet == "primary" {
				// If no primary hosts are up and the records are set to primary hosts, update the records to the backup hosts

				// If the number of current records is equal to the number of backup hosts, update the records
				if len(records) == len(hostSet.Backup.Hosts) {
					logrus.Info("Updating records to backup IPs")
					for i, record := range records {
						record.Content = hostSet.Backup.Hosts[i]
						record.Proxied = &args.ProxyRecords
						err = cloudflareApi.UpdateDNSRecord(ctx, zoneId, record.ID, record)
						checkNilErr(err)
					}
				} else {
					// If the number of current records is not equal to the number of backup hosts, delete all records and create new ones
					logrus.Info("Deleting records and creating new ones")
					for _, record := range records {
						err = cloudflareApi.DeleteDNSRecord(ctx, zoneId, record.ID)
						checkNilErr(err)
					}
					for _, ip := range hostSet.Backup.Hosts {
						record := cloudflare.DNSRecord{
							Type:    "A",
							Name:    recordName,
							Content: ip,
							TTL:     1,
							Proxied: &args.ProxyRecords,
						}
						_, err = cloudflareApi.CreateDNSRecord(ctx, zoneId, record)
						checkNilErr(err)
					}
				}
			} else if ipSet == "unknown" {
				// If the record is set to an unknown IP set, don't do antyhing
				logrus.Info("Record is set to an unknown IP set, doing nothing")
			}
		} else {
			logrus.Info("No primary or backup hosts specified, doing nothing")
		}
		// If loop is set to false, exit the program
		if !args.LoopProgram {
			os.Exit(0)
		} else {
			// Sleep for 1 second
			time.Sleep(time.Duration(time.Duration(1).Seconds()))

			// Reload the configuration and .env file to check for changes
			godotenv.Load()
			arg.MustParse(&args)

		}
	}
}

func ping(host string) bool {
	pinger, err := probing.NewPinger(host)
	checkNilErr(err)
	pinger.Count = 3
	pinger.Timeout = 5 * time.Second
	pinger.SetPrivileged(true)
	// Both Windows and the Docker image work with privileged pings by default
	for {
		err = pinger.Run() // Blocks until finished.
		if err == nil {
			logrus.Debug("Ping sent successfully")
			break
		} else if err.Error() == "listen ip4:icmp : socket: operation not permitted" && runtime.GOOS == "linux" {
			logrus.
				WithField("error", err).
				WithField("help", "Run as root. For more information, check https://github.com/prometheus-community/pro-bing#linux").
				Warn("Privileged ping failed, attempting unprivileged ping")
			pinger.SetPrivileged(false)
			// No break here, so it will try again with unprivileged ping
		} else if err.Error() == "socket: permission denied" && runtime.GOOS == "linux" {
			logrus.
				WithField("error", err).
				WithField("help", "Unprivileged pings are disabled. To enable, run \"sudo sysctl -w net.ipv4.ping_group_range=\"0 2147483647\"\" For more information, check https://github.com/prometheus-community/pro-bing#linux").
				Fatal("Unprivileged ping failed") // Fatal because this is the last attempt at a ping

			break // This is unreachable, but the compiler doesn't know that
		} else {
			checkNilErr(err)
			break
		}
	}
	stats := pinger.Statistics() // get send/receive/duplicate/rtt stats
	// Check if the server is online
	if stats.PacketsRecv > 0 {
		logrus.Infof("Host %v is online", host)
		return true
	} else {
		logrus.Infof("Host %v is offline", host)
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
		logrus.
			WithField("error", err).
			Error("Something happened!")
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
