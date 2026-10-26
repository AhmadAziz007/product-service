package model

import "time"

const (
	InventoryReservationStatusReserved  = "RESERVED"
	InventoryReservationStatusCommitted = "COMMITTED"
	InventoryReservationStatusReleased  = "RELEASED"
)

type InventoryReservation struct {
	ID        int64     `gorm:"primaryKey"`
	OrderCode string    `gorm:"column:order_code;unique;not null;size:64"`
	ProductID int64     `gorm:"column:product_id;not null;index"`
	Quantity  int       `gorm:"column:quantity;not null"`
	Status    string    `gorm:"column:status;not null;size:20;index"`
	ExpiresAt time.Time `gorm:"column:expires_at;not null;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
