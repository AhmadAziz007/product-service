package handlers

import (
	"ecommerce/product-service/config"
	"ecommerce/product-service/internal/adapter"
	"ecommerce/product-service/internal/adapter/handlers/request"
	"ecommerce/product-service/internal/adapter/handlers/response"
	"ecommerce/product-service/internal/core/domain/entity"
	"ecommerce/product-service/internal/core/service"
	"ecommerce/product-service/utils/conv"
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type CartHandlerInterface interface {
	AddToCart(c echo.Context) error
	GetCart(c echo.Context) error
	RemoveFromCart(c echo.Context) error
	RemoveAllCart(c echo.Context) error
	ClearCartInternal(c echo.Context) error
}

type cartHandler struct {
	CartService    service.CartServiceInterface
	ProductService service.ProductServiceInterface
	Config         *config.Config
}

func (ch *cartHandler) ClearCartInternal(c echo.Context) error {
	if c.Request().Header.Get("X-API-Gateway") != "true" || ch.Config.App.GatewaySecretKey == "" ||
		c.Request().Header.Get("X-Gateway-Secret") != ch.Config.App.GatewaySecretKey {
		return c.JSON(http.StatusUnauthorized, response.DefaultResponse{Message: "unauthorized service request"})
	}
	var req struct {
		BuyerID int64 `json:"buyer_id"`
	}
	if err := c.Bind(&req); err != nil || req.BuyerID <= 0 {
		return c.JSON(http.StatusBadRequest, response.DefaultResponse{Message: "valid buyer_id is required"})
	}
	if err := ch.CartService.RemoveAllCart(c.Request().Context(), req.BuyerID); err != nil {
		return c.JSON(http.StatusInternalServerError, response.DefaultResponse{Message: err.Error()})
	}
	return c.JSON(http.StatusOK, response.DefaultResponse{Message: "cart cleared"})
}

// AddToCart implements [CartHandlerInterface].
func (ch *cartHandler) AddToCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		request     = request.CartRequest{}
		jwtUserData = entity.JwtUserData{}
	)

	if err := c.Bind(&request); err != nil {
		log.Errorf("[CartHandler-1] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(request); err != nil {
		log.Errorf("[CartHandler-2] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-3] AddToCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	if err := json.Unmarshal([]byte(user), &jwtUserData); err != nil {
		resp.Message = "invalid session data"
		return c.JSON(http.StatusUnauthorized, resp)
	}

	userID := jwtUserData.UserID
	product, err := ch.ProductService.GetByID(ctx, request.ProductID)
	if err != nil {
		resp.Message = err.Error()
		resp.Data = nil
		if err.Error() == "404" {
			return c.JSON(http.StatusNotFound, resp)
		}
		return c.JSON(http.StatusInternalServerError, resp)
	}
	if product.Status != entity.Product_Approved {
		resp.Message = "only approved products can be added to cart"
		resp.Data = nil
		return c.JSON(http.StatusForbidden, resp)
	}

	reqEntity := entity.CartItem{
		ProductID: request.ProductID,
		Quantity:  request.Quantity,
		CreatedBy: &jwtUserData.UserID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = ch.CartService.AddToCart(ctx, userID, reqEntity)
	if err != nil {
		log.Errorf("[CartHandler-5] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "success add to cart"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// GetCart implements [CartHandlerInterface].
func (ch *cartHandler) GetCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		respList    = []response.CartResponse{}
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-1] GetCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-2] GetCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	userID := jwtUserData.UserID
	items, err := ch.CartService.GetCartByUserID(ctx, userID)
	if err != nil {
		log.Errorf("[CartHandler-3] GetCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	for _, item := range items {
		product, err := ch.ProductService.GetByID(ctx, item.ProductID)
		if err != nil {
			log.Errorf("[CartHandler-4] GetCart: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)
		}

		respList = append(respList, response.CartResponse{
			ID:            item.ProductID,
			ProductName:   product.Name,
			ProductImage:  product.Image,
			ProductStatus: product.Status,
			SalePrice:     int64(product.SalePrice),
			Quantity:      item.Quantity,
			Unit:          product.Unit,
			Weight:        int64(product.Weight),
			CreatedBy:     item.CreatedBy,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
		})
	}

	resp.Message = "success get cart"
	resp.Data = respList
	return c.JSON(http.StatusOK, resp)
}

// RemoveAllCart implements [CartHandlerInterface].
func (ch *cartHandler) RemoveAllCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-1] RemoveAllCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-2] RemoveAllCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	userID := jwtUserData.UserID

	err = ch.CartService.RemoveAllCart(ctx, userID)
	if err != nil {
		log.Errorf("[CartHandler-3] RemoveAllCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// RemoveFromCart implements [CartHandlerInterface].
func (ch *cartHandler) RemoveFromCart(c echo.Context) error {
	var (
		resp        = response.DefaultResponse{}
		ctx         = c.Request().Context()
		jwtUserData = entity.JwtUserData{}
	)

	user := c.Get("user").(string)
	if user == "" {
		log.Errorf("[CartHandler-1] RemoveFromCart: %s", "data token not found")
		resp.Message = "data token not found"
		resp.Data = nil
		return c.JSON(http.StatusNotFound, resp)
	}
	err := json.Unmarshal([]byte(user), &jwtUserData)
	if err != nil {
		log.Errorf("[CartHandler-2] RemoveFromCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}
	userID := jwtUserData.UserID
	productID := c.QueryParam("product_id")
	if productID == "" {
		log.Errorf("[CartHandler-3] RemoveFromCart: %s", "product_id is required")
		resp.Message = "product_id is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	prodID, err := conv.StringToInt64(productID)
	if err != nil {
		log.Errorf("[CartHandler-4] RemoveFromCart: %v", err)
		resp.Message = "invalid product_id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = ch.CartService.RemoveFromCart(ctx, userID, prodID)
	if err != nil {
		log.Errorf("[CartHandler-5] RemoveFromCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}
	resp.Message = "success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

func NewCartHandler(e *echo.Echo, cfg *config.Config, cartService service.CartServiceInterface, productService service.ProductServiceInterface) CartHandlerInterface {
	cartHandler := &cartHandler{
		CartService:    cartService,
		ProductService: productService,
		Config:         cfg,
	}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg)
	authGroup := e.Group("/auth", mid.CheckToken())
	authGroup.POST("/cart", cartHandler.AddToCart)
	authGroup.GET("/cart", cartHandler.GetCart)
	authGroup.DELETE("/cart", cartHandler.RemoveFromCart)
	authGroup.DELETE("/cart/all", cartHandler.RemoveAllCart)
	e.POST("/internal/cart/clear", cartHandler.ClearCartInternal)
	return cartHandler
}
