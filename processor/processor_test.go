package processor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"transfer-sms-notifier/config"
	"transfer-sms-notifier/oracle"
	"transfer-sms-notifier/smsbulk"
	"transfer-sms-notifier/store"
)

func testDB(t *testing.T) (*store.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &store.DB{DB: db}, mock
}

func TestProcessRecords_dryRun(t *testing.T) {
	st, mock := testDB(t)
	mock.ExpectQuery("SELECT COUNT").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	mock.ExpectExec("INSERT INTO processed_transfers").WillReturnResult(sqlmock.NewResult(1, 1))

	cfg := config.Config{
		SMSSendingEnabled: false,
		SendToReceivers:   true,
		SendToSenders:     true,
		ReceiverSMSHeader: "Telesom",
		ReceiverSMSContent: "Recv {{transferId}}",
		SenderSMSHeader: "Telesom",
		SenderSMSContent: "Send {{transferId}}",
		SMSClientID: "transfer_notifier",
	}
	proc := New(cfg, st, nil)
	rec := oracle.Record{Data: map[string]interface{}{
		"ID":                      int64(1),
		"TRANSFERID":              "15,862,087,576",
		"SENDERSUBSCRIPTIONID":    "252639339979",
		"RECEIVERSUBSCRIPTIONID":  "252634872297",
	}}
	stats := proc.ProcessRecords(context.Background(), []oracle.Record{rec})
	if stats.Processed != 1 || stats.Receiver.DryRun != 1 || stats.Sender.DryRun != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessRecords_liveSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send-single" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Fatalf("missing api key")
		}
		var body smsbulk.SendRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Number == "" || body.Header == "" || body.Content == "" {
			t.Fatalf("body=%+v", body)
		}
		_ = json.NewEncoder(w).Encode(smsbulk.SendResponse{
			Success: true, TaskID: "t1", BatchID: body.BatchID, Status: "pending",
		})
	}))
	defer srv.Close()

	st, mock := testDB(t)
	mock.ExpectQuery("SELECT COUNT").WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	mock.ExpectExec("INSERT INTO processed_transfers").WillReturnResult(sqlmock.NewResult(1, 1))

	cfg := config.Config{
		SMSSendingEnabled: true,
		SendToReceivers:   true,
		SendToSenders:     false,
		ReceiverSMSHeader: "Telesom",
		ReceiverSMSContent: "Recv {{transferId}}",
		SMSClientID:       "transfer_notifier",
	}
	sms := smsbulk.NewClient(smsbulk.Config{BaseURL: srv.URL, APIKey: "test-key"})
	proc := New(cfg, st, sms)
	rec := oracle.Record{Data: map[string]interface{}{
		"ID":                     int64(2),
		"TRANSFERID":             "999",
		"RECEIVERSUBSCRIPTIONID": "252634872297",
	}}
	stats := proc.ProcessRecords(context.Background(), []oracle.Record{rec})
	if stats.Receiver.Sent != 1 {
		t.Fatalf("stats=%+v", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
