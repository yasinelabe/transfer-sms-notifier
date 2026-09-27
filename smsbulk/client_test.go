package smsbulk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendSingle_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "key1" {
			t.Fatal("bad key")
		}
		_ = json.NewEncoder(w).Encode(SendResponse{Success: true, BatchID: "b1", TaskID: "t1"})
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL, APIKey: "key1"})
	resp, err := c.SendSingle(context.Background(), SendRequest{
		Number: "252633022408", Header: "Telesom", Content: "hi", BatchID: "b1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Success || resp.TaskID != "t1" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestSendSingle_errorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	c := NewClient(Config{BaseURL: srv.URL, APIKey: "key1"})
	_, err := c.SendSingle(context.Background(), SendRequest{Number: "252633022408", Header: "T", Content: "c"})
	if err == nil {
		t.Fatal("expected error")
	}
}
