package service

import (
	"context"
	"ecommerce/product-service/internal/adapter/message"
	"ecommerce/product-service/internal/adapter/repository"
	"ecommerce/product-service/internal/core/domain/entity"
	"errors"

	"github.com/labstack/gommon/log"
)

type ProductServiceInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) error
	Update(ctx context.Context, req entity.ProductEntity) error
	UpdateStatus(ctx context.Context, productID int64, status string, updatedBy int64) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

var ErrInvalidProductStatusTransition = errors.New("invalid product status transition")

type productService struct {
	repo              repository.ProductRepositoryInterface
	publisherRabbitMQ message.PublishRabbitMQInterface
	repoCat           repository.CategoryRepositoryInterface
}

// SearchProducts implements ProductServiceInterface.
func (p *productService) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.SearchProducts(ctx, query)
}

// Create implements ProductServiceInterface.
func (p *productService) Create(ctx context.Context, req entity.ProductEntity) error {
	productID, err := p.repo.Create(ctx, req)
	if err != nil {
		log.Errorf("[ProductService-1] Create: %v", err)
		return err
	}

	getProductByID, err := p.GetByID(ctx, productID)
	if err != nil {
		log.Errorf("[ProductService-2] Create: %v", err)
	}

	if err := p.publisherRabbitMQ.PublishProductToQueue(*getProductByID); err != nil {
		log.Errorf("[ProductService-3] Create: %v", err)
	}

	return nil
}

// Delete implements ProductServiceInterface.
func (p *productService) Delete(ctx context.Context, productID int64) error {
	err := p.repo.Delete(ctx, productID)
	if err != nil {
		log.Errorf("[ProductService-1] Delete: %v", err)
		return err
	}

	if err := p.publisherRabbitMQ.DeleteProductFromQueue(productID); err != nil {
		log.Errorf("[ProductService-2] Delete: %v", err)
	}

	return nil
}

// GetAll implements ProductServiceInterface.
func (p *productService) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	return p.repo.GetAll(ctx, query)
}

// GetByID implements ProductServiceInterface.
func (p *productService) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	result, err := p.repo.GetByID(ctx, productID)
	if err != nil {
		log.Errorf("[ProductService-1] GetByID: %v", err)
		return nil, err
	}

	resultCat, err := p.repoCat.GetBySlug(ctx, result.CategorySlug)
	if err != nil {
		log.Errorf("[ProductService-2] GetByID: %v", err)
		return nil, err
	}
	if resultCat == nil {
		return nil, errors.New("category not found")
	}
	result.CategoryName = resultCat.Name
	return result, nil
}

// Update implements ProductServiceInterface.
func (p *productService) Update(ctx context.Context, req entity.ProductEntity) error {
	err := p.repo.Update(ctx, req)
	if err != nil {
		log.Errorf("[ProductService-1] Update: %v", err)
		return err
	}

	getProductByID, err := p.GetByID(ctx, req.ID)
	if err != nil {
		log.Errorf("[ProductService-2] Update: %v", err)
	}

	if err := p.publisherRabbitMQ.PublishProductToQueue(*getProductByID); err != nil {
		log.Errorf("[ProductService-3] Update: %v", err)
	}

	return nil
}

func (p *productService) UpdateStatus(ctx context.Context, productID int64, status string, updatedBy int64) error {
	product, err := p.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if !canTransitionProductStatus(product.Status, status) {
		return ErrInvalidProductStatusTransition
	}

	if err := p.repo.UpdateStatus(ctx, productID, status, updatedBy); err != nil {
		return err
	}

	product.Status = status
	product.UpdatedBy = &updatedBy
	for i := range product.Child {
		product.Child[i].Status = status
		product.Child[i].UpdatedBy = &updatedBy
	}
	if err := p.publisherRabbitMQ.PublishProductToQueue(*product); err != nil {
		log.Errorf("[ProductService-3] UpdateStatus: %v", err)
	}

	return nil
}

func canTransitionProductStatus(currentStatus, nextStatus string) bool {
	// Selecting the current status is a no-op and should not be rejected.
	if currentStatus == nextStatus {
		return true
	}

	switch currentStatus {
	case entity.Product_Approved:
		// An approved product can be sent back for review or rejected.
		return nextStatus == entity.Product_Need_Approval ||
			nextStatus == entity.Product_Need_Revision ||
			nextStatus == entity.Product_Rejected
	case entity.Product_Need_Approval:
		return nextStatus == entity.Product_Approved ||
			nextStatus == entity.Product_Need_Revision ||
			nextStatus == entity.Product_Rejected
	case entity.Product_Rejected:
		return nextStatus == entity.Product_Need_Revision
	case entity.Product_Need_Revision:
		return nextStatus == entity.Product_Need_Approval
	default:
		return false
	}
}

func NewProductService(repo repository.ProductRepositoryInterface, publisherRabbitMQ message.PublishRabbitMQInterface, repoCat repository.CategoryRepositoryInterface) ProductServiceInterface {
	return &productService{repo: repo, publisherRabbitMQ: publisherRabbitMQ, repoCat: repoCat}
}
