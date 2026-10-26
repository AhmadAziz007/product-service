package entity

import "time"

type CategoryEntity struct {
	ID          int64           `json:"id"`
	ParentID    *int64          `json:"parent_id"`
	Name        string          `json:"name"`
	Icon        string          `json:"icon"`
	Status      string          `json:"status"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Products    []ProductEntity `json:"products"`
	CreatedBy   *int64          `json:"created_by"`
	UpdatedBy   *int64          `json:"updated_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type QueryStringEntity struct {
	Search    string
	Page      int
	Limit     int
	OrderBy   string
	OrderType string
}
