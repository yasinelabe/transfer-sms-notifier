package processor

import (
	"context"
	"fmt"
	"log"
	"time"

	"transfer-sms-notifier/config"
	"transfer-sms-notifier/oracle"
	"transfer-sms-notifier/smsbulk"
	"transfer-sms-notifier/store"
)

type Processor struct {
	cfg    config.Config
	db     *store.DB
	sms    *smsbulk.Client
}

func New(cfg config.Config, db *store.DB, sms *smsbulk.Client) *Processor {
	return &Processor{cfg: cfg, db: db, sms: sms}
}

type SideStats struct {
	Sent    int
	DryRun  int
	Failed  int
	Skipped int
}

type TickStats struct {
	Processed int
	Sender    SideStats
	Receiver  SideStats
}

func (p *Processor) ProcessRecords(ctx context.Context, records []oracle.Record) TickStats {
	var stats TickStats
	for _, rec := range records {
		if err := p.processOne(ctx, rec, &stats); err != nil {
			log.Printf("[processor] transfer_id=%s error: %v", oracle.NormalizeTransferID(mustString(rec, "TRANSFERID")), err)
		} else {
			stats.Processed++
		}
	}
	return stats
}

func (p *Processor) processOne(ctx context.Context, rec oracle.Record, stats *TickStats) error {
	transferID := oracle.NormalizeTransferID(mustString(rec, "TRANSFERID"))
	if transferID == "" {
		return fmt.Errorf("invalid transfer id")
	}
	exists, err := p.db.ExistsTransferID(ctx, transferID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	oracleID := rec.ID()
	sender := oracle.NormalizeMSISDN(mustString(rec, "SENDERSUBSCRIPTIONID"))
	receiver := oracle.NormalizeMSISDN(mustString(rec, "RECEIVERSUBSCRIPTIONID"))
	createdDate := formatCreatedDate(rec)

	vars := TemplateVars{
		TransferID:  transferID,
		Sender:      sender,
		Receiver:    receiver,
		CreatedDate: createdDate,
	}

	row := &store.ProcessedTransfer{
		OracleID:       oracleID,
		TransferID:     transferID,
		SenderMSISDN:   sender,
		ReceiverMSISDN: receiver,
	}

	if p.cfg.SendToReceivers {
		p.handleSide(ctx, "receiver", receiver, p.cfg.ReceiverSMSHeader, RenderTemplate(p.cfg.ReceiverSMSContent, vars),
			fmt.Sprintf("xfer-%s-receiver", transferID), row, &row.ReceiverStatus, &row.ReceiverBatchID, &row.ReceiverTaskID, &stats.Receiver)
	} else {
		row.ReceiverStatus = store.StatusDisabled
	}

	if p.cfg.SendToSenders {
		p.handleSide(ctx, "sender", sender, p.cfg.SenderSMSHeader, RenderTemplate(p.cfg.SenderSMSContent, vars),
			fmt.Sprintf("xfer-%s-sender", transferID), row, &row.SenderStatus, &row.SenderBatchID, &row.SenderTaskID, &stats.Sender)
	} else {
		row.SenderStatus = store.StatusDisabled
	}

	if err := p.db.InsertProcessed(ctx, row); err != nil {
		if store.IsDuplicate(err) {
			return nil
		}
		return err
	}
	return nil
}

func (p *Processor) handleSide(ctx context.Context, side, msisdn, header, content, batchID string,
	row *store.ProcessedTransfer, statusOut, batchOut, taskOut *string, stats *SideStats) {
	if msisdn == "" {
		*statusOut = store.StatusSkipped
		stats.Skipped++
		log.Printf("[processor] transfer_id=%s side=%s skipped: invalid msisdn", row.TransferID, side)
		return
	}
	if !p.cfg.SMSSendingEnabled {
		*statusOut = store.StatusDryRun
		*batchOut = batchID
		stats.DryRun++
		log.Printf("[processor] dry_run transfer_id=%s side=%s number=%s header=%q content=%q", row.TransferID, side, msisdn, header, content)
		return
	}
	resp, err := p.sms.SendSingle(ctx, smsbulk.SendRequest{
		Number:   msisdn,
		Header:   header,
		Content:  content,
		ClientID: p.cfg.SMSClientID,
		BatchID:  batchID,
		Priority: 5,
	})
	if err != nil {
		*statusOut = store.StatusFailed
		*batchOut = batchID
		stats.Failed++
		log.Printf("[processor] transfer_id=%s side=%s send failed: %v", row.TransferID, side, err)
		return
	}
	*statusOut = store.StatusSent
	*batchOut = resp.BatchID
	if resp.BatchID == "" {
		*batchOut = batchID
	}
	*taskOut = resp.TaskID
	stats.Sent++
	log.Printf("[processor] transfer_id=%s side=%s sent batchId=%s taskId=%s", row.TransferID, side, *batchOut, resp.TaskID)
}

func mustString(rec oracle.Record, key string) string {
	s, _ := rec.GetString(key)
	return s
}

func formatCreatedDate(rec oracle.Record) string {
	if t, ok := rec.GetTime("CREATEDDATE"); ok && !t.IsZero() {
		return t.Format(time.RFC3339)
	}
	s, ok := rec.GetString("CREATEDDATE")
	if ok {
		return s
	}
	return ""
}
