package app

import (
	"context"
	"ecommerce/product-service/config"
	"ecommerce/product-service/internal/adapter/handlers"
	"ecommerce/product-service/internal/adapter/message"
	"ecommerce/product-service/internal/adapter/repository"
	"ecommerce/product-service/internal/adapter/storage"
	"ecommerce/product-service/internal/core/service"
	"ecommerce/product-service/utils/validator"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	middlewareGateway "ecommerce/product-service/internal/middleware"
)

func RunServer() {
	cfg := config.NewConfig()
	db, err := cfg.ConnectionPostgres()
	if err != nil {
		log.Fatalf("[RunServer-1] %v", err)
		return
	}

	elasticInit, err := cfg.InitElasticsearch()
	if err != nil {
		log.Fatalf("[RunServer-2] %v", err)
		return
	}

	storageHandler := storage.NewSupabase(cfg)
	publisherRabbitMQ := message.NewPublishRabbitMQ(cfg)

	categoryRepo := repository.NewCategoryRepository(db.DB)
	productRepo := repository.NewProductRepository(db.DB, elasticInit)
	inventoryRepo := repository.NewInventoryRepository(db.DB)
	cartRepo := repository.NewCartRedisRepository(cfg.NewRedisClient())

	categoryService := service.NewCategoryService(categoryRepo)
	productService := service.NewProductService(productRepo, publisherRabbitMQ, categoryRepo)
	inventoryService := service.NewInventoryService(inventoryRepo)
	cartService := service.NewCartService(cartRepo)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middlewareGateway.GatewayValidationMiddleware())

	customValidator := validator.NewValidator()
	en.RegisterDefaultTranslations(customValidator.Validator, customValidator.Translator)
	e.Validator = customValidator

	e.GET("/api/check", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	handlers.NewCategoryHandler(e, categoryService, cfg)
	handlers.NewProductHandler(e, cfg, productService)
	handlers.NewInventoryHandler(e, cfg, inventoryService)
	handlers.NewUploadImage(e, cfg, storageHandler)
	handlers.NewCartHandler(e, cfg, cartService, productService)

	workerCtx, stopInventoryWorker := context.WithCancel(context.Background())
	go runInventoryExpirationWorker(workerCtx, inventoryService)

	go func() {
		if cfg.App.AppPort == "" {
			cfg.App.AppPort = os.Getenv("APP_PORT")
		}

		err = e.Start(":" + cfg.App.AppPort)
		if err != nil {
			log.Fatalf("[RunServer-2] %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	<-quit
	stopInventoryWorker()

	log.Print("[RunServer-3] Shutting down server of 5 second...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	e.Shutdown(ctx)
}

func runInventoryExpirationWorker(ctx context.Context, inventoryService service.InventoryServiceInterface) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			released, err := inventoryService.ReleaseExpired(ctx, now, 100)
			if err != nil {
				log.Errorf("[InventoryExpirationWorker] Failed to release expired reservations: %v", err)
				continue
			}
			if released > 0 {
				log.Infof("[InventoryExpirationWorker] Released %d expired stock reservations", released)
			}
		}
	}
}
