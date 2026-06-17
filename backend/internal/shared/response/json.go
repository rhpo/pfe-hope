package response

import (
	"github.com/gofiber/fiber/v3"

	"pfe-backend/internal/shared/apperror"
)

type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type PaginatedData struct {
	Items   any `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func OK(c fiber.Ctx, data any) error {
	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func Created(c fiber.Ctx, data any) error {
	return c.Status(fiber.StatusCreated).JSON(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func NoContent(c fiber.Ctx) error {
	return c.Status(fiber.StatusNoContent).Send(nil)
}

func OKPaginated(c fiber.Ctx, items any, total, page, perPage int) error {
	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Success: true,
		Data: PaginatedData{
			Items:   items,
			Total:   total,
			Page:    page,
			PerPage: perPage,
		},
	})
}

func Error(c fiber.Ctx, err error) error {
	if appErr, ok := err.(*apperror.Error); ok {
		return c.Status(appErr.StatusCode()).JSON(ErrorResponse{
			Success: false,
			Error:   appErr.Message,
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Success: false,
		Error:   "Erreur interne du serveur",
	})
}

func ValidationError(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func Unauthorized(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func Forbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func NotFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func Conflict(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
		Success: false,
		Error:   message,
	})
}
