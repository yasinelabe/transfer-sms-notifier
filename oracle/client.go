package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

type Config struct {
	User       string
	Password   string
	ConnString string
}

type Client struct {
	db *sql.DB
}

type PollResult struct {
	Records   []Record
	LastID    int64
	FetchedAt time.Time
}

func NewClient(cfg Config) (*Client, error) {
	dsn, err := buildDSN(cfg.User, cfg.Password, cfg.ConnString)
	if err != nil {
		return nil, fmt.Errorf("oracle dsn: %w", err)
	}
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("open oracle: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var one int
	if err := db.QueryRowContext(ctx, "SELECT 1 FROM DUAL").Scan(&one); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping oracle: %w", err)
	}
	return &Client{db: db}, nil
}

func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *Client) Poll(ctx context.Context, lastID int64, batchSize int) (*PollResult, error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	q := fmt.Sprintf(`SELECT ID, CREATEDDATE, TRANSFERID, SENDERSUBSCRIPTIONID, RECEIVERSUBSCRIPTIONID
FROM VW_V
WHERE ID > :1
ORDER BY ID ASC
FETCH FIRST %d ROWS ONLY`, batchSize)

	rows, err := c.db.QueryContext(ctx, q, lastID)
	if err != nil {
		return nil, fmt.Errorf("query VW_V: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &PollResult{FetchedAt: time.Now()}
	var maxID int64
	for rows.Next() {
		dest := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range dest {
			ptrs[i] = &dest[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		rec := Record{Data: make(map[string]interface{}, len(cols))}
		for i, col := range cols {
			rec.Data[strings.ToUpper(col)] = dest[i]
		}
		result.Records = append(result.Records, rec)
		if id := rec.ID(); id > maxID {
			maxID = id
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result.LastID = maxID
	return result, nil
}
