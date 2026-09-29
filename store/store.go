package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

const (
	StatusSent     = "sent"
	StatusFailed   = "failed"
	StatusSkipped  = "skipped"
	StatusDisabled = "disabled"
	StatusDryRun   = "dry_run"
)

type DB struct {
	*sql.DB
}

type ProcessedTransfer struct {
	OracleID         int64
	TransferID       string
	SenderMSISDN     string
	ReceiverMSISDN   string
	SenderBatchID    string
	ReceiverBatchID  string
	SenderTaskID     string
	ReceiverTaskID   string
	SenderStatus     string
	ReceiverStatus   string
}

func New(ctx context.Context, dsn string) (*DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	d := &DB{db}
	if err := d.migrate(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) migrate(ctx context.Context) error {
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS service_state (
		id TINYINT PRIMARY KEY,
		poll_watermark BIGINT NOT NULL DEFAULT 0,
		initialized TINYINT(1) NOT NULL DEFAULT 0,
		updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
	)`); err != nil {
		return err
	}
	if _, err := d.ExecContext(ctx, `INSERT IGNORE INTO service_state (id, poll_watermark, initialized) VALUES (1, 0, 0)`); err != nil {
		return err
	}
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS processed_transfers (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		oracle_id BIGINT NOT NULL DEFAULT 0,
		transfer_id VARCHAR(64) NOT NULL DEFAULT '',
		sender_msisdn VARCHAR(32) NOT NULL DEFAULT '',
		receiver_msisdn VARCHAR(32) NOT NULL DEFAULT '',
		sender_sms_batch_id VARCHAR(128) NOT NULL DEFAULT '',
		receiver_sms_batch_id VARCHAR(128) NOT NULL DEFAULT '',
		sender_sms_task_id VARCHAR(64) NOT NULL DEFAULT '',
		receiver_sms_task_id VARCHAR(64) NOT NULL DEFAULT '',
		sender_sms_status VARCHAR(16) NOT NULL DEFAULT '',
		receiver_sms_status VARCHAR(16) NOT NULL DEFAULT '',
		created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
		UNIQUE KEY uk_transfer_id (transfer_id),
		INDEX idx_created_at (created_at)
	)`); err != nil {
		return err
	}
	_, err := d.ExecContext(ctx, `ALTER TABLE processed_transfers ADD UNIQUE KEY uk_transfer_id (transfer_id)`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate key name") {
		return err
	}
	return nil
}

func (d *DB) GetState(ctx context.Context) (watermark int64, initialized bool, err error) {
	err = d.QueryRowContext(ctx, `SELECT poll_watermark, initialized FROM service_state WHERE id = 1`).
		Scan(&watermark, &initialized)
	return
}

func (d *DB) SetWatermark(ctx context.Context, watermark int64, initialized bool) error {
	initVal := 0
	if initialized {
		initVal = 1
	}
	_, err := d.ExecContext(ctx, `UPDATE service_state SET poll_watermark = ?, initialized = ? WHERE id = 1`, watermark, initVal)
	return err
}

func (d *DB) ExistsTransferID(ctx context.Context, transferID string) (bool, error) {
	if transferID == "" {
		return false, nil
	}
	var n int
	err := d.QueryRowContext(ctx, `SELECT COUNT(1) FROM processed_transfers WHERE transfer_id = ?`, transferID).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (d *DB) InsertProcessed(ctx context.Context, p *ProcessedTransfer) error {
	_, err := d.ExecContext(ctx, `INSERT INTO processed_transfers (
		oracle_id, transfer_id, sender_msisdn, receiver_msisdn,
		sender_sms_batch_id, receiver_sms_batch_id,
		sender_sms_task_id, receiver_sms_task_id,
		sender_sms_status, receiver_sms_status
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.OracleID, p.TransferID, p.SenderMSISDN, p.ReceiverMSISDN,
		p.SenderBatchID, p.ReceiverBatchID,
		p.SenderTaskID, p.ReceiverTaskID,
		p.SenderStatus, p.ReceiverStatus,
	)
	return err
}

func (d *DB) UpdateProcessed(ctx context.Context, p *ProcessedTransfer) error {
	_, err := d.ExecContext(ctx, `UPDATE processed_transfers SET
		sender_sms_batch_id = ?, receiver_sms_batch_id = ?,
		sender_sms_task_id = ?, receiver_sms_task_id = ?,
		sender_sms_status = ?, receiver_sms_status = ?
		WHERE oracle_id = ?`,
		p.SenderBatchID, p.ReceiverBatchID,
		p.SenderTaskID, p.ReceiverTaskID,
		p.SenderStatus, p.ReceiverStatus,
		p.OracleID,
	)
	return err
}

func IsDuplicate(err error) bool {
	if err == nil {
		return false
	}
	// MySQL errno 1062 (duplicate key) without importing the driver here.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "1062")
}
