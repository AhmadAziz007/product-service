package entity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type CartItem struct {
	ProductID int64     `json:"product_id"`
	Quantity  int64     `json:"quantity"`
	CreatedBy *int64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (item *CartItem) UnmarshalJSON(data []byte) error {
	var value struct {
		ProductID int64           `json:"product_id"`
		Quantity  int64           `json:"quantity"`
		CreatedBy *int64          `json:"created_by"`
		CreatedAt json.RawMessage `json:"created_at"`
		UpdatedAt json.RawMessage `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	createdAt, err := parseCartTime(value.CreatedAt)
	if err != nil {
		return fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := parseCartTime(value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("parse updated_at: %w", err)
	}

	*item = CartItem{
		ProductID: value.ProductID,
		Quantity:  value.Quantity,
		CreatedBy: value.CreatedBy,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	return nil
}

func parseCartTime(raw json.RawMessage) (time.Time, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return time.Time{}, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return time.Time{}, err
	}
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		parsed, err = time.ParseInLocation(layout, strings.TrimSpace(value), time.Local)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, err
}
