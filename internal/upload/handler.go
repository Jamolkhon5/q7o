package upload

import (
	"q7o/internal/common/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) UploadFile(c *fiber.Ctx) error {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return response.BadRequest(c, "No file uploaded")
	}

	// Upload file
	result, err := h.service.UploadAvatar(file)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	return response.Success(c, result)
}

// Middleware to limit file size
func FileSizeLimit(maxSize int64) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == "POST" || c.Method() == "PUT" {
			if c.Get("Content-Length") != "" {
				contentLength := c.Context().Request.Header.ContentLength()
				if int64(contentLength) > maxSize {
					return response.BadRequest(c, "File too large")
				}
			}
		}
		return c.Next()
	}
}