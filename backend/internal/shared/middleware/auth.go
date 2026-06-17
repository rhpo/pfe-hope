package middleware

import (
	"strings"

	"pfe-backend/internal/config"
	"pfe-backend/internal/shared/response"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func AuthRequired(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Unauthorized(c, "Authentification requise")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return response.Unauthorized(c, "Format d'authentification invalide")
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return response.Unauthorized(c, "Token invalide ou expiré")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.Unauthorized(c, "Token invalide")
		}

		profileIDFloat, _ := claims["sub"].(float64)
		profileID := int64(profileIDFloat)
		role, _ := claims["role"].(string)

		c.Locals("profile_id", profileID)
		c.Locals("role", role)

		return c.Next()
	}
}

func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		role := GetRole(c)
		for _, allowed := range allowedRoles {
			if role == allowed {
				return c.Next()
			}
		}
		return response.Forbidden(c, "Accès non autorisé pour ce rôle")
	}
}

func GetProfileID(c fiber.Ctx) int64 {
	id, _ := c.Locals("profile_id").(int64)
	return id
}

func GetRole(c fiber.Ctx) string {
	role, _ := c.Locals("role").(string)
	return role
}
