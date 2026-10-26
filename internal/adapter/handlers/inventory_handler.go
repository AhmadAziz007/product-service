package handlers

import (
	"net/http"
	"time"

	"ecommerce/product-service/config"
	"ecommerce/product-service/internal/adapter/handlers/response"
	"ecommerce/product-service/internal/core/service"

	"github.com/labstack/echo/v4"
)

const inventoryReservationSafetyTTL = 26 * time.Hour

type inventoryHandler struct {
	service service.InventoryServiceInterface
	cfg     *config.Config
}

func (h *inventoryHandler) Reserve(c echo.Context) error {
	if !h.isTrustedServiceRequest(c) {
		return c.JSON(http.StatusUnauthorized, response.DefaultResponse{Message: "unauthorized service request"})
	}
	var req struct {
		OrderCode string `json:"order_code"`
		ProductID int64  `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	if err := c.Bind(&req); err != nil || req.OrderCode == "" || req.ProductID <= 0 || req.Quantity <= 0 {
		return c.JSON(http.StatusBadRequest, response.DefaultResponse{Message: "valid order_code, product_id, and quantity are required"})
	}
	if err := h.service.Reserve(c.Request().Context(), req.OrderCode, req.ProductID, req.Quantity, time.Now().Add(inventoryReservationSafetyTTL)); err != nil {
		return c.JSON(http.StatusConflict, response.DefaultResponse{Message: err.Error()})
	}
	return c.JSON(http.StatusOK, response.DefaultResponse{Message: "stock reserved"})
}

func (h *inventoryHandler) Commit(c echo.Context) error {
	return h.withOrderCode(c, func(orderCode string) error {
		if err := h.service.Commit(c.Request().Context(), orderCode); err != nil {
			return c.JSON(http.StatusConflict, response.DefaultResponse{Message: err.Error()})
		}
		return c.JSON(http.StatusOK, response.DefaultResponse{Message: "stock reservation committed"})
	})
}

func (h *inventoryHandler) Release(c echo.Context) error {
	return h.withOrderCode(c, func(orderCode string) error {
		if err := h.service.Release(c.Request().Context(), orderCode); err != nil {
			return c.JSON(http.StatusInternalServerError, response.DefaultResponse{Message: err.Error()})
		}
		return c.JSON(http.StatusOK, response.DefaultResponse{Message: "stock reservation released"})
	})
}

func (h *inventoryHandler) withOrderCode(c echo.Context, action func(string) error) error {
	if !h.isTrustedServiceRequest(c) {
		return c.JSON(http.StatusUnauthorized, response.DefaultResponse{Message: "unauthorized service request"})
	}
	var req struct {
		OrderCode string `json:"order_code"`
	}
	if err := c.Bind(&req); err != nil || req.OrderCode == "" {
		return c.JSON(http.StatusBadRequest, response.DefaultResponse{Message: "order_code is required"})
	}
	return action(req.OrderCode)
}

func (h *inventoryHandler) isTrustedServiceRequest(c echo.Context) bool {
	return c.Request().Header.Get("X-API-Gateway") == "true" && h.cfg.App.GatewaySecretKey != "" &&
		c.Request().Header.Get("X-Gateway-Secret") == h.cfg.App.GatewaySecretKey
}

func NewInventoryHandler(e *echo.Echo, cfg *config.Config, inventoryService service.InventoryServiceInterface) {
	handler := &inventoryHandler{service: inventoryService, cfg: cfg}
	e.POST("/internal/inventory/reserve", handler.Reserve)
	e.POST("/internal/inventory/commit", handler.Commit)
	e.POST("/internal/inventory/release", handler.Release)
}
