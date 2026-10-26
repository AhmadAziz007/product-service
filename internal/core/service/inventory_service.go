package service

import (
	"context"
	"time"

	"ecommerce/product-service/internal/adapter/repository"
)

type InventoryServiceInterface interface {
	Reserve(ctx context.Context, orderCode string, productID int64, quantity int, expiresAt time.Time) error
	Commit(ctx context.Context, orderCode string) error
	Release(ctx context.Context, orderCode string) error
	ReleaseExpired(ctx context.Context, now time.Time, limit int) (int, error)
}

type inventoryService struct {
	repo repository.InventoryRepositoryInterface
}

func (s *inventoryService) Reserve(ctx context.Context, orderCode string, productID int64, quantity int, expiresAt time.Time) error {
	return s.repo.Reserve(ctx, orderCode, productID, quantity, expiresAt)
}

func (s *inventoryService) Commit(ctx context.Context, orderCode string) error {
	return s.repo.Commit(ctx, orderCode)
}

func (s *inventoryService) Release(ctx context.Context, orderCode string) error {
	return s.repo.Release(ctx, orderCode)
}

func (s *inventoryService) ReleaseExpired(ctx context.Context, now time.Time, limit int) (int, error) {
	return s.repo.ReleaseExpired(ctx, now, limit)
}

func NewInventoryService(repo repository.InventoryRepositoryInterface) InventoryServiceInterface {
	return &inventoryService{repo: repo}
}
