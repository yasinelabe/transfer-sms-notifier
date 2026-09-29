# Transfer SMS Notifier

Poll-only background worker (no REST API). Reads new rows from Oracle view `VW_V` and sends configurable SMS notifications to senders and/or receivers via smsbulk `POST /send-single`.

## Run

```bash
cp example.env .env
# edit .env

go run .
# or: go build -o transfer-sms-notifier && ./transfer-sms-notifier
```

Cross-compile for Linux from Windows (pure-Go Oracle driver, no CGO):

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o transfersmsnotifier .
```

Stop with Ctrl+C or SIGTERM.

## Configuration

All settings are in `.env`. Restart the process after changes.

| Variable | Description |
|----------|-------------|
| `ORACLE_USER`, `ORACLE_PASSWORD`, `ORACLE_CONN_STRING` | Same Oracle DB as autosupport |
| `MYSQL_DSN` | MySQL for watermark + audit |
| `POLL_INTERVAL_SECONDS` | Poll interval (default 30) |
| `POLL_BATCH_SIZE` | Max rows per poll (default 100) |
| `PROCESS_ON_FIRST_POLL` | `true` = process existing rows on first run; `false` = skip to watermark only |
| `SMS_SENDING_ENABLED` | `false` = dry-run (log only); `true` = live SMS |
| `SEND_TO_RECEIVERS` / `SEND_TO_SENDERS` | Which audiences to notify |
| `RECEIVER_SMS_HEADER` / `RECEIVER_SMS_CONTENT` | Receiver SMS (separate header + body) |
| `SENDER_SMS_HEADER` / `SENDER_SMS_CONTENT` | Sender SMS |
| `SMS_CLIENT_ID` | smsbulk clientId |
| `SMSBULK_BASE_URL` | e.g. `http://localhost:8080` |
| `SMSBULK_API_KEY` | smsbulk `SMS_SINGLE_API_KEY` |

Template placeholders: `{{transferId}}`, `{{sender}}`, `{{receiver}}`, `{{createdDate}}`

## Testing workflow

1. Set `SMS_SENDING_ENABLED=false`
2. Run service and watch logs for `dry_run` lines (number, header, content)
3. Check `processed_transfers` in MySQL (`sender_sms_status` / `receiver_sms_status` = `dry_run`)
4. Set `SMS_SENDING_ENABLED=true`, configure smsbulk URL/key, restart

## Oracle view

Expected columns: `ID`, `CREATEDDATE`, `TRANSFERID`, `SENDERSUBSCRIPTIONID`, `RECEIVERSUBSCRIPTIONID`

## MySQL tables

- `service_state` — poll watermark (`poll_watermark` stores the last seen numeric **transfer ID**)
- `processed_transfers` — dedup + SMS audit per transfer (unique on `transfer_id`)

If upgrading from an older build that watermarked by Oracle row `ID`, reset the watermark:

```sql
UPDATE service_state SET poll_watermark = 0, initialized = 0 WHERE id = 1;
```

Then restart (use `PROCESS_ON_FIRST_POLL=true` only if you want to backfill).

## Logs

Each poll logs: `records`, `sent_*`, `dry_run_*`, `failed_*`, `last_transfer_id`, `sms_enabled`.
