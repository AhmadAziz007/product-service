package adapter

import (
	"ecommerce/product-service/config"
	"ecommerce/product-service/internal/adapter/handlers/response"
	"ecommerce/product-service/internal/core/domain/entity"
	"encoding/json"
	"net/http"

	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type MiddlewareAdapterInterface interface {
	CheckToken() echo.MiddlewareFunc
	RequireRole(roleName string) echo.MiddlewareFunc
}

type middlewareAdapter struct {
	cfg *config.Config
}

func (m *middlewareAdapter) RequireRole(roleName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			resp := response.DefaultResponse{}
			sessionJSON, ok := c.Get("user").(string)
			if !ok || sessionJSON == "" {
				resp.Message = "missing or invalid token"
				return c.JSON(http.StatusUnauthorized, resp)
			}

			session := entity.JwtUserData{}
			if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
				log.Errorf("[MiddlewareAdapter-5] RequireRole: %v", err)
				resp.Message = "invalid session data"
				return c.JSON(http.StatusUnauthorized, resp)
			}

			if session.RoleName != roleName {
				resp.Message = "only " + roleName + " can access this endpoint"
				return c.JSON(http.StatusForbidden, resp)
			}

			return next(c)
		}
	}
}

// CheckToken implements MiddlewareAdapterInterface.
func (m *middlewareAdapter) CheckToken() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			respErr := response.DefaultResponse{}
			redisConn := config.NewConfig().NewRedisClient()
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Errorf("[MiddlewareAdapter-1] CheckToken: %s", "missing or invalid token")
				respErr.Message = "missing or invalid token"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}

				return []byte(m.cfg.App.JWTSecretKey), nil
			})
			if err != nil {
				log.Errorf("[MiddlewareAdapter-2] CheckToken: %s", err.Error())
				respErr.Message = err.Error()
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			getSession, err := redisConn.Get(c.Request().Context(), tokenString).Result()
			if err != nil || len(getSession) == 0 {
				log.Errorf("[MiddlewareAdapter-3] CheckToken: %s", err.Error())
				respErr.Message = err.Error()
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			jwtUserData := entity.JwtUserData{}
			err = json.Unmarshal([]byte(getSession), &jwtUserData)
			if err != nil {
				log.Errorf("[MiddlewareAdapter-4] CheckToken: %v", err)
				respErr.Message = err.Error()
				respErr.Data = nil
				return c.JSON(http.StatusInternalServerError, respErr)
			}

			path := c.Request().URL.Path
			segments := strings.Split(strings.Trim(path, "/"), "/")
			if jwtUserData.RoleName == "Customer" && segments[0] == "admin" {
				log.Infof("[MiddlewareAdapter-5] CheckToken: %s", "customer cannot access admin routes")
				respErr.Message = "customer cannot access admin routes"
				respErr.Data = nil
				return c.JSON(http.StatusForbidden, respErr)
			}

			c.Set("user", getSession)
			return next(c)
		}
	}
}

func NewMiddlewareAdapter(cfg *config.Config) MiddlewareAdapterInterface {
	return &middlewareAdapter{
		cfg: cfg,
	}
}
