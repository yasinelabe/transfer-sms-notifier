package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"transfer-sms-notifier/config"
	"transfer-sms-notifier/oracle"
	"transfer-sms-notifier/processor"
	"transfer-sms-notifier/smsbulk"
	"transfer-sms-notifier/store"
)

func main() {
	cfg := config.Load()
	cfg.Validate()

	log.Printf("transfer-sms-notifier starting sms_enabled=%v send_receivers=%v send_senders=%v poll_interval=%ds batch_size=%d process_first_poll=%v",
		cfg.SMSSendingEnabled, cfg.SendToReceivers, cfg.SendToSenders, cfg.PollIntervalSecs, cfg.PollBatchSize, cfg.ProcessFirstPoll)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := store.New(ctx, cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	defer db.Close()

	ora, err := oracle.NewClient(oracle.Config{
		User:       cfg.OracleUser,
		Password:   cfg.OraclePassword,
		ConnString: cfg.OracleConnString,
	})
	if err != nil {
		log.Fatalf("oracle: %v", err)
	}
	defer ora.Close()

	var smsClient *smsbulk.Client
	if cfg.SMSSendingEnabled {
		smsClient = smsbulk.NewClient(smsbulk.Config{
			BaseURL: cfg.SMSBulkBaseURL,
			APIKey:  cfg.SMSBulkAPIKey,
		})
	}

	proc := processor.New(cfg, db, smsClient)

	watermark, initialized, err := db.GetState(ctx)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	log.Printf("state watermark_transfer_id=%d initialized=%v", watermark, initialized)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	interval := time.Duration(cfg.PollIntervalSecs) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	runPoll := func() {
		pollCtx, pollCancel := context.WithTimeout(ctx, 2*time.Minute)
		defer pollCancel()

		result, err := ora.Poll(pollCtx, watermark, cfg.PollBatchSize)
		if err != nil {
			log.Printf("[poll] error: %v", err)
			return
		}

		recordCount := len(result.Records)
		if !initialized {
			if recordCount == 0 {
				initialized = true
				if err := db.SetWatermark(pollCtx, watermark, true); err != nil {
					log.Printf("[poll] set watermark: %v", err)
				}
				log.Printf("[poll] init complete (empty view) watermark_transfer_id=%d", watermark)
				return
			}
			if !cfg.ProcessFirstPoll {
				newMark := result.LastTransferID
				if newMark > watermark {
					watermark = newMark
				}
				initialized = true
				if err := db.SetWatermark(pollCtx, watermark, true); err != nil {
					log.Printf("[poll] set watermark: %v", err)
				}
				log.Printf("[poll] init skipped processing records=%d watermark_transfer_id=%d", recordCount, watermark)
				return
			}
		}

		var stats processor.TickStats
		if recordCount > 0 {
			stats = proc.ProcessRecords(pollCtx, result.Records)
			if result.LastTransferID > watermark {
				watermark = result.LastTransferID
			}
			if !initialized {
				initialized = true
			}
			if err := db.SetWatermark(pollCtx, watermark, initialized); err != nil {
				log.Printf("[poll] set watermark: %v", err)
			}
		}

		log.Printf("[poll] records=%d processed=%d sent_receiver=%d sent_sender=%d dry_run_receiver=%d dry_run_sender=%d failed_receiver=%d failed_sender=%d last_transfer_id=%d sms_enabled=%v",
			recordCount, stats.Processed,
			stats.Receiver.Sent, stats.Sender.Sent,
			stats.Receiver.DryRun, stats.Sender.DryRun,
			stats.Receiver.Failed, stats.Sender.Failed,
			watermark, cfg.SMSSendingEnabled)
	}

	log.Printf("polling every %s", interval)
	runPoll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			log.Println("shutting down...")
			cancel()
			return
		case <-ticker.C:
			runPoll()
		}
	}
}
