package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	OracleUser       string
	OraclePassword   string
	OracleConnString string
	PollIntervalSecs int
	PollBatchSize    int
	ProcessFirstPoll bool

	MySQLDSN string

	SMSSendingEnabled bool
	SendToReceivers     bool
	SendToSenders       bool
	ReceiverSMSHeader   string
	ReceiverSMSContent  string
	SenderSMSHeader     string
	SenderSMSContent    string

	SMSClientID    string
	SMSBulkBaseURL string
	SMSBulkAPIKey  string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("config: no .env file (%v), using environment", err)
	}

	return Config{
		OracleUser:       strings.TrimSpace(os.Getenv("ORACLE_USER")),
		OraclePassword:   os.Getenv("ORACLE_PASSWORD"),
		OracleConnString: strings.TrimSpace(os.Getenv("ORACLE_CONN_STRING")),
		PollIntervalSecs: envInt("POLL_INTERVAL_SECONDS", 30),
		PollBatchSize:    envInt("POLL_BATCH_SIZE", 100),
		ProcessFirstPoll: envBool("PROCESS_ON_FIRST_POLL"),

		MySQLDSN: strings.TrimSpace(os.Getenv("MYSQL_DSN")),

		SMSSendingEnabled: envBool("SMS_SENDING_ENABLED"),
		SendToReceivers:     envBoolDefault("SEND_TO_RECEIVERS", true),
		SendToSenders:       envBoolDefault("SEND_TO_SENDERS", true),
		ReceiverSMSHeader:   strings.TrimSpace(os.Getenv("RECEIVER_SMS_HEADER")),
		ReceiverSMSContent:  os.Getenv("RECEIVER_SMS_CONTENT"),
		SenderSMSHeader:     strings.TrimSpace(os.Getenv("SENDER_SMS_HEADER")),
		SenderSMSContent:    os.Getenv("SENDER_SMS_CONTENT"),

		SMSClientID:    strings.TrimSpace(envDefault("SMS_CLIENT_ID", "transfer_notifier")),
		SMSBulkBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("SMSBULK_BASE_URL")), "/"),
		SMSBulkAPIKey:  os.Getenv("SMSBULK_API_KEY"),
	}
}

func (c Config) Validate() {
	if c.OracleUser == "" || c.OraclePassword == "" || c.OracleConnString == "" {
		log.Fatal("config: ORACLE_USER, ORACLE_PASSWORD, and ORACLE_CONN_STRING are required")
	}
	if c.MySQLDSN == "" {
		log.Fatal("config: MYSQL_DSN is required")
	}
	if c.SMSSendingEnabled {
		if c.SMSBulkBaseURL == "" || c.SMSBulkAPIKey == "" {
			log.Fatal("config: SMSBULK_BASE_URL and SMSBULK_API_KEY are required when SMS_SENDING_ENABLED=true")
		}
	}
	if !c.SendToReceivers && !c.SendToSenders {
		log.Fatal("config: at least one of SEND_TO_RECEIVERS or SEND_TO_SENDERS must be true")
	}
	if c.SendToReceivers && (c.ReceiverSMSHeader == "" || c.ReceiverSMSContent == "") {
		log.Fatal("config: RECEIVER_SMS_HEADER and RECEIVER_SMS_CONTENT are required when SEND_TO_RECEIVERS=true")
	}
	if c.SendToSenders && (c.SenderSMSHeader == "" || c.SenderSMSContent == "") {
		log.Fatal("config: SENDER_SMS_HEADER and SENDER_SMS_CONTENT are required when SEND_TO_SENDERS=true")
	}
	if len(c.ReceiverSMSHeader) > 11 {
		log.Fatal("config: RECEIVER_SMS_HEADER must be at most 11 characters")
	}
	if len(c.SenderSMSHeader) > 11 {
		log.Fatal("config: SENDER_SMS_HEADER must be at most 11 characters")
	}
}

func envDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("config: invalid %s: %q", key, v)
	}
	return n
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func envBoolDefault(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return envBool(key)
}
