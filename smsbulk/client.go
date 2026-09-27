package smsbulk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type Client struct {
	http *http.Client
	cfg  Config
}

type SendRequest struct {
	Number   string `json:"number"`
	Header   string `json:"header"`
	Content  string `json:"content"`
	ClientID string `json:"clientId"`
	BatchID  string `json:"batchId"`
	Priority int    `json:"priority,omitempty"`
}

type SendResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	TaskID   string `json:"taskId"`
	BatchID  string `json:"batchId"`
	ClientID string `json:"clientId"`
	Status   string `json:"status"`
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) SendSingle(ctx context.Context, req SendRequest) (*SendResponse, error) {
	if strings.TrimSpace(c.cfg.BaseURL) == "" {
		return nil, fmt.Errorf("smsbulk base URL not configured")
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/send-single", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.cfg.APIKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusConflict {
		return nil, fmt.Errorf("batchId already exists: %s", req.BatchID)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("smsbulk status %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var out SendResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("invalid smsbulk response: %w", err)
	}
	return &out, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
