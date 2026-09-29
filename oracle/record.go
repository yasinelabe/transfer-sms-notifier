package oracle

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Data map[string]interface{}
}

func (r *Record) GetInt64(key string) (int64, bool) {
	if v, ok := r.Data[key]; ok {
		switch val := v.(type) {
		case int64:
			return val, true
		case int:
			return int64(val), true
		case float64:
			return int64(val), true
		case string:
			if i, err := strconv.ParseInt(val, 10, 64); err == nil {
				return i, true
			}
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return int64(f), true
			}
		default:
			str := fmt.Sprintf("%v", val)
			if i, err := strconv.ParseInt(str, 10, 64); err == nil {
				return i, true
			}
			if f, err := strconv.ParseFloat(str, 64); err == nil {
				return int64(f), true
			}
		}
	}
	return 0, false
}

func (r *Record) GetString(key string) (string, bool) {
	if v, ok := r.Data[key]; ok {
		if val, ok := v.(string); ok {
			return val, true
		}
		if v != nil {
			return fmt.Sprintf("%v", v), true
		}
	}
	return "", false
}

func (r *Record) GetTime(key string) (time.Time, bool) {
	if v, ok := r.Data[key]; ok {
		if val, ok := v.(time.Time); ok {
			return val, true
		}
	}
	return time.Time{}, false
}

func (r *Record) ID() int64 {
	id, _ := r.GetInt64("ID")
	return id
}

func ParseTransferIDNumeric(input string) int64 {
	norm := NormalizeTransferID(input)
	if norm == "" {
		return 0
	}
	n, err := strconv.ParseInt(norm, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func (r *Record) TransferIDNumeric() int64 {
	s, _ := r.GetString("TRANSFERID")
	return ParseTransferIDNumeric(s)
}

func (r *Record) String() string {
	var parts []string
	for k, v := range r.Data {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

func NormalizeMSISDN(input string) string {
	var digits strings.Builder
	for _, ch := range input {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	s := digits.String()
	if len(s) < 9 {
		return ""
	}
	return s
}

func NormalizeTransferID(input string) string {
	var digits strings.Builder
	for _, ch := range input {
		if ch >= '0' && ch <= '9' {
			digits.WriteRune(ch)
		}
	}
	return digits.String()
}
