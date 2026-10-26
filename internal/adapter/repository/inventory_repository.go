package repository

import (
	"context"
	"errors"
	"time"

	"ecommerce/product-service/internal/core/domain/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryRepositoryInterface interface {
	Reserve(ctx context.Context, orderCode string, productID int64, quantity int, expiresAt time.Time) error
	Commit(ctx context.Context, orderCode string) error
	Release(ctx context.Context, orderCode string) error
	ReleaseExpired(ctx context.Context, now time.Time, limit int) (int, error)
}

type inventoryRepository struct {
	db *gorm.DB
}

func (r *inventoryRepository) Reserve(ctx context.Context, orderCode string, productID int64, quantity int, expiresAt time.Time) error {
	if orderCode == "" || productID <= 0 || quantity <= 0 {
		return errors.New("order_code, product_id, and quantity must be valid")
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.InventoryReservation
		err := tx.Where("order_code = ?", orderCode).First(&existing).Error
		if err == nil {
			if existing.ProductID != productID || existing.Quantity != quantity {
				return errors.New("order_code is already reserved for a different item")
			}
			if existing.Status == model.InventoryReservationStatusCommitted ||
				(existing.Status == model.InventoryReservationStatusReserved && existing.ExpiresAt.After(time.Now())) {
				return nil
			}
			if existing.Status == model.InventoryReservationStatusReserved {
				return errors.New("inventory reservation has expired")
			}
			return errors.New("inventory reservation was already released")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		result := tx.Model(&model.Product{}).
			Where("id = ? AND stock >= ? AND status = ?", productID, quantity, "APPROVED").
			UpdateColumn("stock", gorm.Expr("stock - ?", quantity))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("insufficient stock or product not found")
		}

		reservation := model.InventoryReservation{
			OrderCode: orderCode,
			ProductID: productID,
			Quantity:  quantity,
			Status:    model.InventoryReservationStatusReserved,
			ExpiresAt: expiresAt,
		}
		return tx.Create(&reservation).Error
	})
	if err == nil {
		return nil
	}
	// Concurrent retries may both observe no reservation. If the other request
	// committed the same reservation first, treat this request as idempotent.
	var existing model.InventoryReservation
	if lookupErr := r.db.WithContext(ctx).Where("order_code = ?", orderCode).First(&existing).Error; lookupErr == nil &&
		existing.ProductID == productID && existing.Quantity == quantity &&
		(existing.Status == model.InventoryReservationStatusCommitted ||
			(existing.Status == model.InventoryReservationStatusReserved && existing.ExpiresAt.After(time.Now()))) {
		return nil
	}
	return err
}

func (r *inventoryRepository) Commit(ctx context.Context, orderCode string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation model.InventoryReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_code = ?", orderCode).First(&reservation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("inventory reservation not found")
			}
			return err
		}
		switch reservation.Status {
		case model.InventoryReservationStatusCommitted:
			return nil
		case model.InventoryReservationStatusReleased:
			return errors.New("inventory reservation has already been released")
		case model.InventoryReservationStatusReserved:
			return tx.Model(&reservation).Update("status", model.InventoryReservationStatusCommitted).Error
		default:
			return errors.New("invalid inventory reservation status")
		}
	})
}

func (r *inventoryRepository) Release(ctx context.Context, orderCode string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation model.InventoryReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_code = ?", orderCode).First(&reservation).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if reservation.Status != model.InventoryReservationStatusReserved {
			return nil
		}

		if err := tx.Model(&model.Product{}).Where("id = ?", reservation.ProductID).
			UpdateColumn("stock", gorm.Expr("stock + ?", reservation.Quantity)).Error; err != nil {
			return err
		}
		return tx.Model(&reservation).Update("status", model.InventoryReservationStatusReleased).Error
	})
}

func (r *inventoryRepository) ReleaseExpired(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	released := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservations []model.InventoryReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND expires_at <= ?", model.InventoryReservationStatusReserved, now).
			Order("expires_at ASC").Limit(limit).Find(&reservations).Error; err != nil {
			return err
		}
		for _, reservation := range reservations {
			if err := tx.Model(&model.Product{}).Where("id = ?", reservation.ProductID).
				UpdateColumn("stock", gorm.Expr("stock + ?", reservation.Quantity)).Error; err != nil {
				return err
			}
			if err := tx.Model(&reservation).Update("status", model.InventoryReservationStatusReleased).Error; err != nil {
				return err
			}
			released++
		}
		return nil
	})
	return released, err
}

func NewInventoryRepository(db *gorm.DB) InventoryRepositoryInterface {
	return &inventoryRepository{db: db}
}
