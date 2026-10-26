package repository

import (
	"context"
	"ecommerce/product-service/internal/core/domain/entity"
	"ecommerce/product-service/internal/core/domain/model"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"strconv"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductRepositoryInterface interface {
	GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	Create(ctx context.Context, req entity.ProductEntity) (int64, error)
	Update(ctx context.Context, req entity.ProductEntity) error
	UpdateStatus(ctx context.Context, productID int64, status string, updatedBy int64) error
	Delete(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
}

type productRepository struct {
	db       *gorm.DB
	esClient *elasticsearch.Client
}

// Delete implements ProductRepositoryInterface.
func (p *productRepository) Delete(ctx context.Context, productID int64) error {
	modelProduct := model.Product{}

	if err := p.db.WithContext(ctx).Preload("Childs").First(&modelProduct, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
		}
		log.Errorf("[ProductRepository-1] Delete: %v", err)
		return err
	}

	if err := p.db.WithContext(ctx).Select("Childs").Delete(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository-2] Delete: %v", err)
		return err
	}

	res, err := p.esClient.Delete(
		"products",
		strconv.Itoa(int(productID)),
		p.esClient.Delete.WithRefresh("true"),
	)
	if err != nil {
		log.Errorf("[ProductRepository-3] Delete: %v", err)
		return err
	}

	defer res.Body.Close()
	log.Infof("[ProductRepository-4] Delete Product Elasticsearch: %d", productID)

	return nil
}

// Update implements ProductRepositoryInterface.
func (p *productRepository) Update(ctx context.Context, req entity.ProductEntity) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		modelProduct := model.Product{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", req.ID).First(&modelProduct).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = errors.New("404")
			}
			log.Errorf("[ProductRepository-1] Update: %v", err)
			return err
		}

		modelProduct.CategorySlug = req.CategorySlug
		modelProduct.ParentID = req.ParentID
		modelProduct.Name = req.Name
		modelProduct.Image = req.Image
		modelProduct.Description = req.Description
		modelProduct.RegulerPrice = req.RegulerPrice
		modelProduct.SalePrice = req.SalePrice
		modelProduct.Unit = req.Unit
		modelProduct.Weight = req.Weight
		modelProduct.Stock = req.Stock
		modelProduct.Variant = req.Variant
		modelProduct.Status = req.Status
		modelProduct.UpdatedBy = req.UpdatedBy
		now := time.Now()
		modelProduct.UpdatedAt = &now

		if err := tx.Save(&modelProduct).Error; err != nil {
			log.Errorf("[ProductRepository-2] Update: %v", err)
			return err
		}

		if len(req.Child) > 0 {
			if err := tx.Where("parent_id = ?", modelProduct.ID).Delete(&model.Product{}).Error; err != nil {
				log.Errorf("[ProductRepository-3] Update: %v", err)
				return err
			}

			modelProductChild := []model.Product{}
			for _, val := range req.Child {
				modelProductChild = append(modelProductChild, model.Product{
					CategorySlug: req.CategorySlug,
					ParentID:     &modelProduct.ID,
					Name:         req.Name,
					Image:        val.Image,
					Description:  req.Description,
					RegulerPrice: val.RegulerPrice,
					SalePrice:    val.SalePrice,
					Unit:         req.Unit,
					Weight:       val.Weight,
					Stock:        val.Stock,
					Variant:      req.Variant,
					Status:       req.Status,
					UpdatedBy:    req.UpdatedBy,
				})
			}

			if err := tx.Create(&modelProductChild).Error; err != nil {
				log.Errorf("[ProductRepository-3] Update: %v", err)
				return err
			}
		}
		return nil
	})
}

func (p *productRepository) UpdateStatus(ctx context.Context, productID int64, status string, updatedBy int64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product model.Product
		if err := tx.Where("id = ? AND parent_id IS NULL", productID).First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("404")
			}
			return err
		}

		return tx.Model(&model.Product{}).
			Where("id = ? OR parent_id = ?", productID, productID).
			Updates(map[string]interface{}{
				"status":     status,
				"updated_by": updatedBy,
				"updated_at": time.Now(),
			}).Error
	})
}

// Create implements ProductRepositoryInterface.
func (p *productRepository) Create(ctx context.Context, req entity.ProductEntity) (int64, error) {
	modelProduct := model.Product{
		CategorySlug: req.CategorySlug,
		ParentID:     req.ParentID,
		Name:         req.Name,
		Image:        req.Image,
		Description:  req.Description,
		RegulerPrice: req.RegulerPrice,
		SalePrice:    req.SalePrice,
		Unit:         req.Unit,
		Weight:       req.Weight,
		Stock:        req.Stock,
		Variant:      req.Variant,
		Status:       req.Status,
		CreatedBy:    req.CreatedBy,
	}

	if err := p.db.Create(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository-1] Create: %v", err)
		return 0, err
	}

	if len(req.Child) > 0 {
		modelProductChild := []model.Product{}
		for _, val := range req.Child {
			modelProductChild = append(modelProductChild, model.Product{
				CategorySlug: req.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         req.Name,
				Image:        val.Image,
				Description:  req.Description,
				RegulerPrice: val.RegulerPrice,
				SalePrice:    val.SalePrice,
				Unit:         req.Unit,
				Weight:       val.Weight,
				Stock:        val.Stock,
				Variant:      req.Variant,
				Status:       req.Status,
				CreatedBy:    req.CreatedBy,
			})
		}

		if err := p.db.Create(&modelProductChild).Error; err != nil {
			log.Errorf("[ProductRepository-2] Create: %v", err)
			return 0, err
		}
	}

	return modelProduct.ID, nil
}

// GetByID implements ProductRepositoryInterface.
func (p *productRepository) GetByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	modelProduct := model.Product{}

	if err := p.db.WithContext(ctx).Preload("Category").First(&modelProduct, "id = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("404")
		}
		log.Errorf("[ProductRepository-1] GetByID: %v", err)
		return nil, err
	}

	modelParent := []model.Product{}
	err := p.db.WithContext(ctx).Preload("Category").Where("parent_id = ?", modelProduct.ID).Find(&modelParent).Error
	if err != nil {
		log.Errorf("[ProductRepository-2] GetByID: %v", err)
		return nil, err
	}

	childEntities := []entity.ProductEntity{}
	for _, val := range modelParent {
		childEntities = append(childEntities, entity.ProductEntity{
			ID:           val.ID,
			CategorySlug: val.CategorySlug,
			ParentID:     val.ParentID,
			Name:         val.Name,
			Image:        val.Image,
			Description:  val.Description,
			RegulerPrice: val.RegulerPrice,
			SalePrice:    val.SalePrice,
			Unit:         val.Unit,
			Weight:       val.Weight,
			Stock:        val.Stock,
			Variant:      val.Variant,
			Status:       val.Status,
			CategoryName: val.Category.Name,
			Child:        childEntities,
			CreatedAt:    val.CreatedAt,
			UpdatedAt:    val.UpdatedAt,
			CreatedBy:    val.CreatedBy,
			UpdatedBy:    val.UpdatedBy,
		})
	}

	return &entity.ProductEntity{
		ID:           modelProduct.ID,
		CategorySlug: modelProduct.CategorySlug,
		ParentID:     modelProduct.ParentID,
		Name:         modelProduct.Name,
		Image:        modelProduct.Image,
		Description:  modelProduct.Description,
		RegulerPrice: modelProduct.RegulerPrice,
		SalePrice:    modelProduct.SalePrice,
		Unit:         modelProduct.Unit,
		Weight:       modelProduct.Weight,
		Stock:        modelProduct.Stock,
		Variant:      modelProduct.Variant,
		Status:       modelProduct.Status,
		CategoryName: modelProduct.Category.Name,
		Child:        childEntities,
		CreatedAt:    modelProduct.CreatedAt,
		UpdatedAt:    modelProduct.UpdatedAt,
		CreatedBy:    modelProduct.CreatedBy,
		UpdatedBy:    modelProduct.UpdatedBy,
	}, nil
}

// GetAll implements ProductRepositoryInterface.
func (p *productRepository) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	from := (query.Page - 1) * query.Limit
	if from < 0 {
		from = 0
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	sortField := "id"
	if query.OrderBy != "" {
		sortField = query.OrderBy
	}

	// Menentukan urutan sorting (asc atau desc)
	sortOrder := "asc"
	if query.OrderType == "desc" {
		sortOrder = "desc"
	}

	var mustQueries []interface{}
	if query.CategorySlug != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{"category_slug.keyword": query.CategorySlug},
		})
	}
	if query.Status != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{"status.keyword": query.Status},
		})
	}

	if query.StartPrice > 0 && query.EndPrice > 0 {
		mustQueries = append(mustQueries, map[string]interface{}{
			"range": map[string]interface{}{
				"reguler_price": map[string]interface{}{"gte": query.StartPrice, "lte": query.EndPrice},
			},
		})
	}

	if query.Search != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query.Search,
				"fields": []string{"name", "description", "category_name"},
			},
		})
	}
	if len(mustQueries) == 0 {
		mustQueries = append(mustQueries, map[string]interface{}{"match_all": map[string]interface{}{}})
	}

	mainQuery, err := json.Marshal(map[string]interface{}{
		"from": from,
		"size": limit,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": mustQueries,
			},
		},
		"sort": []interface{}{map[string]interface{}{sortField: sortOrder}},
	})
	if err != nil {
		return nil, 0, 0, err
	}

	res, err := p.esClient.Search(
		p.esClient.Search.WithContext(ctx),
		p.esClient.Search.WithIndex("products"),
		p.esClient.Search.WithBody(strings.NewReader(string(mainQuery))),
		p.esClient.Search.WithPretty(),
	)

	if err != nil {
		log.Printf("Error searching Elasticsearch: %s", err)
		return nil, 0, 0, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, 0, 0, fmt.Errorf("elasticsearch search failed with status %s", res.Status())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source entity.ProductEntity `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("Error decoding response: %s", err)
		return nil, 0, 0, err
	}

	totalData := result.Hits.Total.Value

	totalPage := 0
	if limit > 0 {
		totalPage = int(math.Ceil(float64(totalData) / float64(limit)))
	}

	products := make([]entity.ProductEntity, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		products = append(products, hit.Source)
	}

	return products, int64(totalData), int64(totalPage), nil
}

// GetAll implements ProductRepositoryInterface.
func (p *productRepository) GetAll(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	modelProducts := []model.Product{}
	var countData int64

	order := fmt.Sprintf("%s %s", query.OrderBy, query.OrderType)
	offset := (query.Page - 1) * query.Limit
	sqlMain := p.db.Preload("Category").
		Where("parent_id IS NULL").
		Where("name ILIKE ? OR description ILIKE ? OR category_slug ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%")
	if query.Status != "" {
		sqlMain = sqlMain.Where("status = ?", query.Status)
	}
	if query.CategorySlug != "" {
		sqlMain = sqlMain.Where("category_slug = ?", query.CategorySlug)
	}

	if query.StartPrice > 0 {
		sqlMain = sqlMain.Where("sale_price >= ?", query.StartPrice)
	}

	if query.EndPrice > 0 {
		sqlMain = sqlMain.Where("sale_price <= ?", query.EndPrice)
	}

	if err := sqlMain.Model(&modelProducts).Count(&countData).Error; err != nil {
		log.Errorf("[ProductRepository-1] GetAll: %v", err)
		return nil, 0, 0, err
	}

	totalPage := int(math.Ceil(float64(countData) / float64(query.Limit)))
	if err := sqlMain.Order(order).Limit(int(query.Limit)).Offset(int(offset)).Find(&modelProducts).Error; err != nil {
		log.Errorf("[ProductRepository-2] GetAll: %v", err)
		return nil, 0, 0, err
	}

	if len(modelProducts) == 0 {
		log.Errorf("[ProductRepository-3] GetAll: %v", "Data not found")
		return nil, 0, 0, errors.New("404")
	}

	respProducts := []entity.ProductEntity{}
	for _, val := range modelProducts {
		respProducts = append(respProducts, entity.ProductEntity{
			ID:           val.ID,
			CategorySlug: val.CategorySlug,
			ParentID:     val.ParentID,
			Name:         val.Name,
			Image:        val.Image,
			Description:  val.Description,
			RegulerPrice: val.RegulerPrice,
			SalePrice:    val.SalePrice,
			Unit:         val.Unit,
			Weight:       val.Weight,
			Stock:        val.Stock,
			Variant:      val.Variant,
			Status:       val.Status,
			CategoryName: val.Category.Name,
			CreatedAt:    val.CreatedAt,
			UpdatedAt:    val.UpdatedAt,
			CreatedBy:    val.CreatedBy,
			UpdatedBy:    val.UpdatedBy,
		})
	}

	return respProducts, countData, int64(totalPage), nil
}

func NewProductRepository(db *gorm.DB, es *elasticsearch.Client) ProductRepositoryInterface {
	return &productRepository{db: db, esClient: es}
}
